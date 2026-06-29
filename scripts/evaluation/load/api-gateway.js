import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  vus: 5,
  duration: "30s",
  thresholds: {
    http_req_duration: ["p(95)<1000"],
    http_req_failed: ["rate<0.05"],
  },
};

const baseURL = __ENV.API_GATEWAY_URL || "http://localhost:8081";

export default function () {
  const response = http.get(`${baseURL}/health`);
  check(response, {
    "gateway health returns success": (r) => r.status >= 200 && r.status < 300,
  });
  sleep(1);
}
