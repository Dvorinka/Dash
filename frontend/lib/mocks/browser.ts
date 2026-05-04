"use client";

import { handlers } from "./handlers";

let installed = false;

export function installMocks() {
  if (installed || typeof window === "undefined") return;

  import("msw/browser").then(({ setupWorker }) => {
    const worker = setupWorker(...handlers);
    worker.start({ onUnhandledRequest: "bypass" });
    installed = true;
    console.log("[MSW] Mock Service Worker installed");
  });
}
