FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-hubspot"]
COPY baton-hubspot /