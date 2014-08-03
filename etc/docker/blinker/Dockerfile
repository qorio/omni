# Blinker Dockerfile
#

# Pull base image.
FROM lab616/openalpr-1.1.0:latest

# Install blinker from build directory
# This assumes that the binary has been copied into the directory where this Dockerfile lives.
COPY blinker /root/
COPY testAuthKey.pub /root/testAuthKey.pub
RUN chmod a+x /root/blinker

# Define mountable directories.
VOLUME ["/data"]

# Define working directory.
WORKDIR /root

# Define default command.
CMD ["/root/blinker", "--logtostderr", "--dir=/data", "--auth_public_key_file=testAuthKey.pub"]

# Expose ports.

EXPOSE 5050
EXPOSE 7070