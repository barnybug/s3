FROM golang:1.3-cross
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -yq \
upx \
--no-install-recommends
ADD build.sh /
CMD /build.sh
