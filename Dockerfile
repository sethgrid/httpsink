FROM docker.sendgrid.net/sendgrid/dev_go-1.7

ADD . /opt/go/src/github.com/sendgrid/httpsink
WORKDIR /opt/go/src/github.com/sendgrid/httpsink

RUN ["bin/test"]
RUN ["bin/build"]

ENV HTTPSINK_PORT=50111\
  HTTPSINK_HOST=0.0.0.0\
  HTTPSINK_CAPACITY=0\
  HTTPSINK_PROXY=\
  HTTPSINK_TTL=5m

EXPOSE $HTTPSINK_PORT

CMD ["./build/httpsink", "--port=$HTTPSINK_PORT", "--host=$HTTPSINK_HOST", "--capacity=$HTTPSINK_CAPACITY", "--proxy=$HTTPSINK_PROXY", "--ttl=$HTTPSINK_TTL"]
