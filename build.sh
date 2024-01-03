docker buildx build \
  --platform "linux/arm64" \
  --output "./dist" \
  --target "artifact" .
