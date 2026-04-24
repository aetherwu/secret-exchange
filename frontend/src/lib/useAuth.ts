"use client";

import { useEffect, useState } from "react";
import { isLoggedIn } from "./api";
import { useRouter } from "next/navigation";

export function useRequireAuth() {
  const router = useRouter();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (!isLoggedIn()) {
      router.replace("/login");
    } else {
      setReady(true);
    }
  }, [router]);

  return ready;
}
