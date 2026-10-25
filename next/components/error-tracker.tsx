"use client";

import { useEffect } from "react";

import { installGlobalErrorHandlers } from "@/lib/error-tracking";

/** Memasang listener error/unhandledrejection global. Tidak merender apa pun. */
export function ErrorTracker() {
  useEffect(() => installGlobalErrorHandlers(), []);
  return null;
}
