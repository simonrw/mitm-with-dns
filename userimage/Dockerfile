FROM alpine

RUN apk update && \
    apk add \
    aws-cli \
    bind-tools \
    curl \
    neofetch

ENV AWS_ACCESS_KEY_ID test
ENV AWS_SECRET_ACCESS_KEY test
ENV AWS_REGION us-east-1

CMD ["sleep", "86400"]
