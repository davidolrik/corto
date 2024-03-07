# Used by goreleaser: the prebuilt, fully self-contained corto binary
# (embedded web UI and migrations) is copied into a minimal base image.
FROM gcr.io/distroless/static-debian12:nonroot

ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/corto /corto

# Containers must bind all interfaces for port mapping to work
ENV CORTO_SERVER_IP=0.0.0.0

EXPOSE 3000

ENTRYPOINT ["/corto"]
CMD ["server"]
