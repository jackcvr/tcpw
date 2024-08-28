FROM alpine:latest

RUN wget -O /usr/local/bin/tcpw https://raw.githubusercontent.com/jackcvr/tcpw/main/tcpw/tcpw-$(uname -m) \
    && chmod +x /usr/local/bin/tcpw