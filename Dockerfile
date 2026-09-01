FROM gcr.io/distroless/static@sha256:f2ea2709ac8db56323cbd7d014277f32cb572d9ea124b0076f7aafe5980678fe

ENTRYPOINT ["/app/submit-patch"]

COPY /dist/submit-patch /app/submit-patch
