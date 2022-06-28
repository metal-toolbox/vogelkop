FROM scratch
COPY vogelkop /vogelkop
ENTRYPOINT ["/vogelkop"]
CMD ["--help"]