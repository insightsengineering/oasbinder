FROM scratch
COPY oasbinder /
WORKDIR /
# hadolint ignore=DL3025
EXPOSE 8080
ENTRYPOINT ["/oasbinder"]
