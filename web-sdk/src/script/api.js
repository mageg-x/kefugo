import axios from "axios";
import { toSdkError } from "./error-codes";

class Api {
  constructor() {
    this.baseURL =
      typeof window !== "undefined" && window.location?.origin
        ? window.location.origin
        : "http://localhost:5300";
    this.userId = null;
    this.api = axios.create({
      baseURL: this.baseURL,
      timeout: 10000,
      headers: {
        "Content-Type": "application/json",
      },
    });
  }

  setBaseURL(url) {
    if (!url) {
      return;
    }
    this.baseURL = url;
    this.api.defaults.baseURL = url;
  }

  setUserId(userId) {
    this.userId = userId;
  }

  async getConfig(appId) {
    try {
      const params = {
        appid: appId,
      };
      
      if (this.userId) {
        params.userid = this.userId;
      }
      
      const response = await this.api.get("/api/v1/config", {
        params: params,
      });
      return response.data;
    } catch (error) {
      console.error("Failed to get config:", error);
      throw toSdkError(error, "Failed to get config");
    }
  }

  async uploadFile({ appId, file, contentType }) {
    if (!appId) {
      throw new Error("app_id required");
    }
    if (!file) {
      throw new Error("file required");
    }
    const formData = new FormData();
    formData.append("app_id", appId);
    if (contentType) {
      formData.append("content_type", contentType);
    }
    formData.append("file", file);

    try {
      const response = await this.api.post("/api/v1/upload", formData, {
        params: {
          app_id: appId,
        },
        headers: {
          "Content-Type": "multipart/form-data",
        },
      });
      return response.data;
    } catch (error) {
      throw toSdkError(error, "Upload failed");
    }
  }

  async getVisitorHistory({ appId, visitorId, limit = 50, before = "" }) {
    if (!appId) {
      throw new Error("app_id required");
    }
    if (!visitorId) {
      throw new Error("visitor_id required");
    }
    try {
      const response = await this.api.get("/api/v1/visitor/history", {
        params: {
          app_id: appId,
          visitor_id: visitorId,
          limit,
          ...(before ? { before } : {}),
        },
      });
      return response.data;
    } catch (error) {
      throw toSdkError(error, "Failed to get history");
    }
  }

  async rateSession({ sid, appId, visitorId, score, comment = "" }) {
    if (!sid || !appId || !visitorId) {
      throw new Error("sid/appId/visitorId required");
    }
    return this.api.post("/api/v1/sessions/rate", {
      sid,
      app_id: String(appId || ""),
      visitor_id: String(visitorId || ""),
      score: Number(score || 0),
      comment: String(comment || ""),
    });
  }
}

export default new Api();
