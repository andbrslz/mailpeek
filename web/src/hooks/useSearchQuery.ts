import { useEffect, useState } from "react";
import { useDebounced } from "./useMessages";

const readQuery = () => new URLSearchParams(window.location.search).get("q") ?? "";

export function useSearchQuery() {
  const [query, setQuery] = useState(readQuery);
  const search = useDebounced(query.trim(), 150);

  useEffect(() => {
    const url = new URL(window.location.href);
    if (search) url.searchParams.set("q", search);
    else url.searchParams.delete("q");
    if (url.href !== window.location.href) window.history.replaceState(null, "", url);
  }, [search]);

  return { query, setQuery, search };
}
