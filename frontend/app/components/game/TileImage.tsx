import Image from "next/image";
import type { ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";

const tileImageCache = new Map<string, string>();

const buildCandidates = (symbol: string) => {
  const trimmed = symbol.trim();
  if (!trimmed) return [];
  const hasExtension = trimmed.endsWith(".svg");
  if (trimmed.startsWith("/")) {
    return [trimmed];
  }
  if (hasExtension) {
    return [`/tiles/${trimmed}`];
  }
  return [`/tiles/${trimmed}.svg`, `/tiles/svg/${trimmed}.svg`];
};

interface TileImageProps {
  symbol: string;
  alt: string;
  width: number;
  height: number;
  className?: string;
  fallback?: ReactNode;
}

export default function TileImage({
  symbol,
  alt,
  width,
  height,
  className,
  fallback = null,
}: TileImageProps) {
  const trimmed = symbol.trim();
  const candidates = useMemo(() => buildCandidates(trimmed), [trimmed]);
  const [index, setIndex] = useState(0);
  const [exhausted, setExhausted] = useState(false);

  useEffect(() => {
    setExhausted(false);
    if (!trimmed) {
      setIndex(0);
      return;
    }
    const cached = tileImageCache.get(trimmed);
    if (cached) {
      const cachedIndex = candidates.indexOf(cached);
      setIndex(cachedIndex >= 0 ? cachedIndex : 0);
      return;
    }
    setIndex(0);
  }, [trimmed, candidates]);

  if (!trimmed || candidates.length === 0) return null;
  if (exhausted) return <>{fallback}</>;
  const src = candidates[index];

  return (
    <Image
      src={src}
      alt={alt}
      width={width}
      height={height}
      className={className}
      onError={() => {
        if (index < candidates.length - 1) {
          setIndex(index + 1);
        } else {
          setExhausted(true);
        }
      }}
      onLoadingComplete={() => {
        tileImageCache.set(trimmed, src);
      }}
    />
  );
}
