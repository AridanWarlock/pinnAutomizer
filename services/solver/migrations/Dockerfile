FROM kukymbr/goose-docker:3.27.0
# Share the kernel file lock with the application; CLI migrations must not
# modify the journal while a solver owns it.
RUN apk add --no-cache util-linux
COPY services/solver/migrations/entrypoint.sh /goose-solver/entrypoint.sh
RUN chmod +x /goose-solver/entrypoint.sh
ENTRYPOINT ["/goose-solver/entrypoint.sh"]
