# OpenALPR dockerfile
# build from release 1.1.0 version tag

FROM ubuntu:14.04

RUN \
apt-get update && \
apt-get -y install libopencv-dev libtesseract-dev git cmake build-essential && \
apt-get -y install liblog4cplus-dev libcurl3-dev uuid-dev wget && \
git clone https://github.com/gyokuro/openalpr.git && \
cd openalpr && \
git checkout tags/v1.1.1 && \
cd src && \
mkdir build && \
cd build && \
cmake -DCMAKE_INSTALL_PREFIX:PATH=/usr .. && \
make && \
make install && \
wget http://easy-clan.com/ski/pics/license_plate.JPG -O lp.jpg && \
alpr lp.jpg
