import { useEffect, useState } from "react";
import { getInfo } from "../api";

export interface ServerInfo {
  smtpAddress: string;
  smtpAuth: boolean;
  uiAuth: boolean;
}

export function useServerInfo(): ServerInfo {
  const [info, setInfo] = useState({ port: 1026, smtpAuth: false, uiAuth: false });
  useEffect(() => {
    let cancelled = false;
    getInfo()
      .then((i) => {
        if (!cancelled) setInfo({ port: i.smtpPort, smtpAuth: i.smtpAuth, uiAuth: i.uiAuth });
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, []);
  return {
    smtpAddress: `${window.location.hostname || "localhost"}:${info.port}`,
    smtpAuth: info.smtpAuth,
    uiAuth: info.uiAuth,
  };
}
