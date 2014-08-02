# Blinker Dockerfile
#

# Pull base image.
FROM ubuntu:14.04

# Install blinker from Circle CI
#RUN \
#  wget -O blinker https://circle-artifacts.com/gh/qorio/omni/226/artifacts/0/tmp/circle-artifacts.QncARRk/linux_amd64/blinker?circle-token=b71701145614b93a382a8e3b5d633ee71c360315 && \
#  chmod a+x blinker

COPY blinker /root/
COPY testAuthKey.pub /root/testAuthKey.pub
RUN chmod a+x blinker

# Define mountable directories.
VOLUME ["/data"]

# Define working directory.
WORKDIR /root

# Define default command.
CMD ["/root/blinker", "--logtostderr", "--dir=/data", "--auth_public_key_file=testAuthKey.pub"]

# Expose ports.

EXPOSE 5050
EXPOSE 7070