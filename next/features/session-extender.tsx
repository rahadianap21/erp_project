// features/session-extender.tsx
"use client";

import { useEffect, useRef } from "react";
import Cookies from "js-cookie";

import {
  buildSessionRefresh,
  sessionCookieOptions,
  sessionExpiry,
} from "./session";

const THROTTLE_MS = 5000;

export default function SessionExtender() {
  const throttleRef = useRef<boolean>(false);

  const refreshCookieLifetime = () => {
    if (throttleRef.current) return;
    throttleRef.current = true;

    setTimeout(() => {
      throttleRef.current = false;
    }, THROTTLE_MS);

    // user_email adalah penanda masih login atau tidak.
    if (!Cookies.get("user_email") || !Cookies.get("user_app")) return;

    const options = sessionCookieOptions(sessionExpiry());

    // buildSessionRefresh menentukan key mana yang benar-benar ada isinya,
    // termasuk user_permissions. Cookie yang tidak ada di-skip agar tidak
    // ditimpa nilai kosong.
    for (const [key, value] of Object.entries(
      buildSessionRefresh(Cookies.get)
    )) {
      Cookies.set(key, value, options);
    }
  };

  useEffect(() => {
    const events = ["mousemove", "mousedown", "keydown", "touchstart"];

    events.forEach((event) => {
      window.addEventListener(event, refreshCookieLifetime);
    });

    return () => {
      events.forEach((event) => {
        window.removeEventListener(event, refreshCookieLifetime);
      });
    };
  }, []);

  return null;
}
