#!/usr/bin/env python3
# Unified local model service for Go integration testing.
# Dependencies:
#   pip install llama-cpp-python fastapi uvicorn pydantic torch transformers
#
# Standard-first routes:
#   /v1/chat/completions
#   /v1/embeddings
#   /v1/rerank
#
# Compatibility aliases:
#   /chat/completions
#   /embeddings
#   /rerank
#   /api/embed
#   /api/rerank

import json
import os
import re
import threading
import time
from typing import List, Optional, Union

import torch
import uvicorn
from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse, StreamingResponse
from llama_cpp import Llama
from pydantic import BaseModel
from transformers import AutoModelForSequenceClassification, AutoTokenizer


# ================== Config ==================
MODEL_DIR = os.environ.get("MODEL_DIR", "/home/steven/models")
CHAT_MODEL = os.environ.get("CHAT_MODEL", "qwen2.5-0.5b-instruct-q8_0.gguf")
EMBED_MODEL = os.environ.get("EMBED_MODEL", "Qwen3-Embedding-0.6B-Q8_0.gguf")
RERANK_MODEL = os.environ.get("RERANK_MODEL", "bge-reranker-base")
RERANK_INSTRUCTION = os.environ.get(
    "RERANK_INSTRUCTION",
    "Given a user query, retrieve relevant passages that answer the query",
)

N_THREADS = int(os.environ.get("N_THREADS", "4"))
CTX_SIZE = int(os.environ.get("CTX_SIZE", "2048"))
EMBED_CTX = int(os.environ.get("EMBED_CTX", "512"))
N_BATCH = int(os.environ.get("N_BATCH", "512"))
CHAT_MAX_TOKENS = int(os.environ.get("CHAT_MAX_TOKENS", "256"))
RERANK_MAX_LENGTH = int(os.environ.get("RERANK_MAX_LENGTH", "512"))

torch.set_num_threads(max(1, N_THREADS))


def resolve_model_path(name: str) -> str:
    raw = (name or "").strip()
    if not raw:
        return raw
    if os.path.isabs(raw):
        return raw
    return os.path.join(MODEL_DIR, raw)


# ================== Load models ==================
print("Loading chat model...")
chat_llm = Llama(
    model_path=resolve_model_path(CHAT_MODEL),
    n_ctx=CTX_SIZE,
    n_threads=N_THREADS,
    n_batch=N_BATCH,
    verbose=False,
)

print("Loading embedding model...")
embed_llm = Llama(
    model_path=resolve_model_path(EMBED_MODEL),
    embedding=True,
    n_ctx=EMBED_CTX,
    n_threads=N_THREADS,
    n_batch=N_BATCH,
    verbose=False,
)

print("Loading rerank model...")
rerank_model_path = resolve_model_path(RERANK_MODEL)
rerank_tokenizer = AutoTokenizer.from_pretrained(rerank_model_path, local_files_only=True)
rerank_model = AutoModelForSequenceClassification.from_pretrained(
    rerank_model_path,
    local_files_only=True,
)
rerank_model.eval()
print("All models loaded.")

chat_lock = threading.Lock()
embed_lock = threading.Lock()
rerank_lock = threading.Lock()
embed_batch_fallback_warned = False


# ================== App ==================
app = FastAPI(title="Local GGUF API")


# ================== Request models ==================
class EmbedRequest(BaseModel):
    input: Union[str, List[str]]
    model: Optional[str] = None


class OllamaEmbedRequest(BaseModel):
    model: Optional[str] = None
    input: Union[str, List[str]]


class ChatMessage(BaseModel):
    role: str
    content: str


class ChatRequest(BaseModel):
    messages: List[ChatMessage]
    model: Optional[str] = None
    stream: Optional[bool] = False


class RerankRequest(BaseModel):
    model: Optional[str] = None
    query: str
    documents: List[str]
    top_n: Optional[int] = None
    debug: Optional[bool] = False


class OllamaRerankRequest(BaseModel):
    model: Optional[str] = None
    query: str
    documents: List[str]
    debug: Optional[bool] = False


# ================== Helpers ==================
def build_prompt(messages: List[ChatMessage]) -> str:
    lines = []
    for msg in messages:
        role = (msg.role or "user").strip().lower()
        content = (msg.content or "").strip()
        if not content:
            continue
        if role == "system":
            lines.append(f"system: {content}")
        elif role == "assistant":
            lines.append(f"assistant: {content}")
        else:
            lines.append(f"user: {content}")
    lines.append("assistant: ")
    return "\n".join(lines)


def do_embed_single(text: str) -> list:
    with embed_lock:
        return embed_llm.embed(text)


def do_embed_many(texts: List[str]) -> List[list]:
    with embed_lock:
        return embed_llm.embed(texts)


def log_timing(endpoint: str, started_at: float, **fields):
    duration_ms = (time.perf_counter() - started_at) * 1000.0
    extras = " ".join(f"{key}={value}" for key, value in fields.items() if value is not None)
    message = f"timing: endpoint={endpoint} duration_ms={duration_ms:.1f}"
    if extras:
        message += " " + extras
    print(message, flush=True)


def normalize_text(text: str) -> str:
    return re.sub(r"\s+", " ", (text or "").strip()).lower()


def normalize_compact(text: str) -> str:
    return re.sub(r"\s+", "", normalize_text(text))


def tokenize_for_overlap(text: str) -> List[str]:
    compact = normalize_text(text)
    if not compact:
        return []
    ascii_words = re.findall(r"[a-z0-9]+", compact)
    cjk_chars = re.findall(r"[\u4e00-\u9fff]", compact)
    return ascii_words + cjk_chars


def extract_query_topic(query: str) -> str:
    topic = normalize_compact(query)
    if not topic:
        return ""
    for pattern in [
        r"^(请问|请介绍一下|介绍一下|请介绍|介绍|说明一下|说明|解释一下|解释)",
        r"^(什么是|啥是|何谓|什么叫|什么叫做|什么叫作)",
        r"(是什么|是啥|有哪些|有哪几种|怎么用|如何使用|如何理解|是什么意思)$",
        r"[？?。！，,、；;：:]",
    ]:
        topic = re.sub(pattern, "", topic)
    return topic.strip()


def lexical_relevance_score(query: str, document: str) -> float:
    q_tokens = tokenize_for_overlap(query)
    d_tokens = tokenize_for_overlap(document)
    if not q_tokens or not d_tokens:
        return 0.0
    q_set = set(q_tokens)
    d_set = set(d_tokens)
    overlap = len(q_set & d_set)
    if overlap <= 0:
        return 0.0
    return overlap / max(len(q_set), 1)


def structural_relevance_score(query: str, document: str) -> float:
    topic = extract_query_topic(query)
    doc_compact = normalize_compact(document)
    if not topic or not doc_compact:
        return 0.0

    score = 0.0
    if topic in doc_compact:
        score += 0.35
    if doc_compact.startswith(topic):
        score += 0.4
    elif f"“{topic}”" in document or f'"{topic}"' in document:
        score += 0.1

    # Definition-style query should prefer docs where the topic is the sentence subject.
    if any(marker in normalize_compact(query) for marker in ["什么是", "是什么", "啥是", "何谓"]):
        if doc_compact.startswith(topic):
            score += 0.15
        elif f"是{topic}的" in doc_compact or f"属于{topic}" in doc_compact:
            score -= 0.1

    return max(0.0, min(1.0, score))


def score_rerank_base(query: str, document: str) -> dict:
    lexical_score = lexical_relevance_score(query, document)
    structural_score = structural_relevance_score(query, document)
    base_score = max(0.0, min(1.0, 0.55 * lexical_score + 0.45 * structural_score))
    topic = extract_query_topic(query)
    debug = {
        "topic": topic,
        "lexical_score": lexical_score,
        "structural_score": structural_score,
        "base_score": base_score,
        "model_score": base_score,
        "model_mode": "cross_encoder",
        "model_logit": None,
        "prompt_tokens": 0,
        "used_model_score": False,
        "fallback_reason": "",
        "final_score": base_score,
    }
    return debug


def score_rerank_batch(query: str, documents: List[str]) -> List[dict]:
    pairs = [[query, doc] for doc in documents]
    base_infos = [score_rerank_base(query, doc) for doc in documents]
    if not pairs:
        return base_infos

    with rerank_lock:
        encoded = rerank_tokenizer(
            pairs,
            padding=True,
            truncation=True,
            max_length=RERANK_MAX_LENGTH,
            return_tensors="pt",
        )
        prompt_tokens = int(encoded["input_ids"].shape[1]) if "input_ids" in encoded else 0
        with torch.no_grad():
            logits = rerank_model(**encoded).logits

    if logits.dim() == 2 and logits.shape[1] == 1:
        scores_tensor = torch.sigmoid(logits[:, 0])
        raw_logits = logits[:, 0]
    elif logits.dim() == 2 and logits.shape[1] >= 2:
        scores_tensor = torch.softmax(logits, dim=1)[:, -1]
        raw_logits = logits[:, -1]
    else:
        flat_logits = logits.reshape(-1)
        scores_tensor = torch.sigmoid(flat_logits)
        raw_logits = flat_logits

    for idx, info in enumerate(base_infos):
        if idx >= len(scores_tensor):
            break
        score = float(scores_tensor[idx].item())
        info["model_score"] = score
        info["model_logit"] = float(raw_logits[idx].item())
        info["prompt_tokens"] = prompt_tokens
        info["used_model_score"] = True
        info["final_score"] = score
    return base_infos


def run_chat_completion(req: ChatRequest) -> dict:
    prompt = build_prompt(req.messages)
    with chat_lock:
        response = chat_llm(
            prompt,
            max_tokens=CHAT_MAX_TOKENS,
            stop=["\nuser:", "\nsystem:"],
            echo=False,
        )
    answer = response["choices"][0]["text"].strip()
    usage = response.get("usage", {})
    return {
        "id": "chatcmpl-local",
        "object": "chat.completion",
        "created": int(time.time()),
        "model": req.model or "local-chat",
        "choices": [
            {
                "index": 0,
                "message": {"role": "assistant", "content": answer},
                "finish_reason": "stop",
            }
        ],
        "usage": {
            "prompt_tokens": usage.get("prompt_tokens", 0),
            "completion_tokens": usage.get("completion_tokens", 0),
            "total_tokens": usage.get("total_tokens", 0),
        },
    }


def stream_chat_completion(req: ChatRequest, started_at: float):
    prompt = build_prompt(req.messages)
    created = int(time.time())
    payload_id = "chatcmpl-local"
    model_name = req.model or "local-chat"
    prompt_chars = sum(len((msg.content or "")) for msg in req.messages)
    answer_parts = []
    chunk_count = 0

    started = {
        "id": payload_id,
        "object": "chat.completion.chunk",
        "created": created,
        "model": model_name,
        "choices": [
            {
                "index": 0,
                "delta": {"role": "assistant"},
                "finish_reason": None,
            }
        ],
    }
    yield f"data: {json.dumps(started, ensure_ascii=False)}\n\n"

    try:
        with chat_lock:
            stream = chat_llm(
                prompt,
                max_tokens=CHAT_MAX_TOKENS,
                stop=["\nuser:", "\nsystem:"],
                echo=False,
                stream=True,
            )
            for piece in stream:
                text = ""
                if piece and piece.get("choices"):
                    text = str(piece["choices"][0].get("text") or "")
                if not text:
                    continue
                answer_parts.append(text)
                chunk_count += 1
                chunk = {
                    "id": payload_id,
                    "object": "chat.completion.chunk",
                    "created": created,
                    "model": model_name,
                    "choices": [
                        {
                            "index": 0,
                            "delta": {"content": text},
                            "finish_reason": None,
                        }
                    ],
                }
                yield f"data: {json.dumps(chunk, ensure_ascii=False)}\n\n"
    finally:
        answer_text = "".join(answer_parts).strip()
        log_timing(
            "chat.completions",
            started_at,
            stream=True,
            messages=len(req.messages),
            prompt_chars=prompt_chars,
            answer_chars=len(answer_text),
            chunks=chunk_count,
        )

    done = {
        "id": payload_id,
        "object": "chat.completion.chunk",
        "created": created,
        "model": model_name,
        "choices": [{"index": 0, "delta": {}, "finish_reason": "stop"}],
    }
    yield f"data: {json.dumps(done, ensure_ascii=False)}\n\n"
    yield "data: [DONE]\n\n"


def build_embeddings_response(req_model: Optional[str], raw_input: Union[str, List[str]]) -> dict:
    global embed_batch_fallback_warned
    texts = raw_input if isinstance(raw_input, list) else [raw_input]
    normalized_texts = [str(text or "") for text in texts]
    data = []
    total_tokens = 0
    embeddings = []
    if normalized_texts:
        try:
            embeddings = do_embed_many(normalized_texts)
        except Exception as exc:
            if not embed_batch_fallback_warned:
                print(f"warn: batch embeddings fallback to single-item mode: {exc}")
                embed_batch_fallback_warned = True
    for i, text in enumerate(normalized_texts):
        embedding = embeddings[i] if i < len(embeddings) else do_embed_single(text)
        data.append({"object": "embedding", "embedding": embedding, "index": i})
        total_tokens += len(text)
    return {
        "object": "list",
        "data": data,
        "model": req_model or "local-embedding",
        "usage": {"prompt_tokens": total_tokens, "total_tokens": total_tokens},
    }


def build_rerank_response(
    req_model: Optional[str],
    query: str,
    documents: List[str],
    top_n: Optional[int],
    debug: bool = False,
) -> dict:
    score_infos = score_rerank_batch(query, documents)
    scored = []
    for i, doc in enumerate(documents):
        score_info = score_infos[i] if i < len(score_infos) else score_rerank_base(query, doc)
        item = {
            "index": i,
            "relevance_score": score_info["final_score"],
            "document": {"text": doc},
        }
        if debug:
            item["debug"] = score_info
        scored.append(item)
    scored.sort(key=lambda x: x["relevance_score"], reverse=True)
    limit = top_n if top_n and top_n > 0 else len(scored)
    return {"results": scored[:limit], "model": req_model or "local-reranker"}


# ================== Health ==================
@app.get("/")
async def root():
    return {"status": "ok", "service": "local-gguf-api"}


@app.get("/health")
async def health():
    return {"status": "ok"}


# ================== Chat routes ==================
@app.post("/v1/chat/completions")
@app.post("/chat/completions")
async def chat_completions(req: ChatRequest):
    started_at = time.perf_counter()
    try:
        if req.stream:
            return StreamingResponse(
                stream_chat_completion(req, started_at),
                media_type="text/event-stream",
                headers={
                    "Cache-Control": "no-cache",
                    "Connection": "keep-alive",
                    "X-Accel-Buffering": "no",
                },
            )
        payload = run_chat_completion(req)
        prompt_chars = sum(len((msg.content or "")) for msg in req.messages)
        answer_chars = len(payload["choices"][0]["message"]["content"])
        log_timing(
            "chat.completions",
            started_at,
            stream=False,
            messages=len(req.messages),
            prompt_chars=prompt_chars,
            answer_chars=answer_chars,
        )
        return JSONResponse(payload)
    except Exception as exc:
        log_timing("chat.completions", started_at, error=type(exc).__name__)
        raise HTTPException(500, detail=str(exc))


# ================== Embedding routes ==================
@app.post("/v1/embeddings")
@app.post("/embeddings")
async def embeddings(req: EmbedRequest):
    started_at = time.perf_counter()
    try:
        payload = build_embeddings_response(req.model, req.input)
        item_count = len(payload.get("data", []))
        input_value = req.input if isinstance(req.input, list) else [req.input]
        input_chars = sum(len(str(item or "")) for item in input_value)
        log_timing(
            "embeddings",
            started_at,
            items=item_count,
            input_chars=input_chars,
        )
        return payload
    except Exception as exc:
        log_timing("embeddings", started_at, error=type(exc).__name__)
        raise HTTPException(500, detail=str(exc))


@app.post("/api/embed")
async def ollama_embed(req: OllamaEmbedRequest):
    try:
        payload = build_embeddings_response(req.model, req.input)
        return {
            "model": payload["model"],
            "embeddings": [item["embedding"] for item in payload["data"]],
        }
    except Exception as exc:
        raise HTTPException(500, detail=str(exc))


# ================== Rerank routes ==================
@app.post("/v1/rerank")
@app.post("/rerank")
async def rerank(req: RerankRequest):
    started_at = time.perf_counter()
    try:
        payload = build_rerank_response(req.model, req.query, req.documents, req.top_n, bool(req.debug))
        log_timing(
            "rerank",
            started_at,
            docs=len(req.documents),
            top_n=req.top_n if req.top_n is not None else len(payload.get("results", [])),
            query_chars=len(req.query or ""),
        )
        return payload
    except Exception as exc:
        log_timing("rerank", started_at, error=type(exc).__name__)
        raise HTTPException(500, detail=str(exc))


@app.post("/api/rerank")
async def ollama_rerank(req: OllamaRerankRequest):
    try:
        payload = build_rerank_response(req.model, req.query, req.documents, None, bool(req.debug))
        result = {
            "model": payload["model"],
            "results": [
                {
                    "index": item["index"],
                    "relevance_score": item["relevance_score"],
                }
                for item in payload["results"]
            ],
        }
        if req.debug:
            for idx, item in enumerate(payload["results"]):
                result["results"][idx]["debug"] = item.get("debug", {})
        return result
    except Exception as exc:
        raise HTTPException(500, detail=str(exc))


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8680)
