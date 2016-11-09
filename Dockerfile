FROM docker.sendgrid.net/sendgrid/dev_go-1.7

ADD . /opt/go/src/github.com/sendgrid/httpsink
WORKDIR /opt/go/src/github.com/sendgrid/httpsink

RUN ["bin/test"]
RUN ["bin/build"]

ENV HTTPSINK_PORT=50111\
  HTTPSINK_INTERFACE=0.0.0.0

EXPOSE 50111

CMD ["./build/httpsink"]
