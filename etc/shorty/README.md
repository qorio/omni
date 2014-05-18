# Shorty Server Configuraton

## Console

[DigitalOcean](https://cloud.digitalocean.com/droplets)

## DNS

[Name.com](https://www.name.com/account/domain/details/qor.io#dns)


## Software

List of sofware

Open Source

- Nginx (all)
- Redis (Shorty)

Proprietary

- shorty
- dasher

For now everything installed as root, in the root directory.


### Update the host

```
apt-get update
apt-get dist-upgrade
apt-get upgrade
```


### Nginx

```
apt-get install nginx
service nginx start
update-rc.d nginx defaults
```

### Redis
```
sudo apt-get install build-essential
sudo apt-get install tcl8.5
wget http://download.redis.io/releases/redis-2.8.9.tar.gz
tar xvfz redis-2.8.9.tar.gz
cd redis-2.8.9/
make
make install
cd utils/
./install_server.sh```
root@sfo-redis-1:~/redis-2.8.9/utils# ./install_server.sh
Welcome to the redis service installer
This script will help you easily set up a running redis server

Please select the redis port for this instance: [6379]
Selecting default: 6379
Please select the redis config file name [/etc/redis/6379.conf]
Selected default - /etc/redis/6379.conf
Please select the redis log file name [/var/log/redis_6379.log]
Selected default - /var/log/redis_6379.log
Please select the data directory for this instance [/var/lib/redis/6379]
Selected default - /var/lib/redis/6379
Please select the redis executable path [/usr/local/bin/redis-server]
Selected config:
Port           : 6379
Config file    : /etc/redis/6379.conf
Log file       : /var/log/redis_6379.log
Data dir       : /var/lib/redis/6379
Executable     : /usr/local/bin/redis-server
Cli Executable : /usr/local/bin/redis-cli
Is this ok? Then press ENTER to go on or Ctrl-C to abort.
Copied /tmp/6379.conf => /etc/init.d/redis_6379
Installing service...
 Adding system startup for /etc/init.d/redis_6379 ...
   /etc/rc0.d/K20redis_6379 -> ../init.d/redis_6379
   /etc/rc1.d/K20redis_6379 -> ../init.d/redis_6379
   /etc/rc6.d/K20redis_6379 -> ../init.d/redis_6379
   /etc/rc2.d/S20redis_6379 -> ../init.d/redis_6379
   /etc/rc3.d/S20redis_6379 -> ../init.d/redis_6379
   /etc/rc4.d/S20redis_6379 -> ../init.d/redis_6379
   /etc/rc5.d/S20redis_6379 -> ../init.d/redis_6379
Success!
Starting Redis server...
Installation successful!
```

### Nginx Setup

From `omni/etc/nginx` in local git repo:

```
scp -r ssl root@107.170.248.96:/etc/nginx
scp shorty.conf root@107.170.248.96:/etc/nginx/sites-available/default0
```
On host:
```
```
service nginx restart
```

### Get Shorty Build

- Builds are available on [Circle CI](https://circleci.com/gh/qorio/omni).
- API token is `b71701145614b93a382a8e3b5d633ee71c360315`
- Append `circle-token=b71701145614b93a382a8e3b5d633ee71c360315` as the `wget` parameter.
- The directory, `/root/shorty` needs to be `chmod 777` so that nginx can talk on the domain sockets.


```
cd
mkdir shorty
chmod 777 shorty
cd shorty
wget https://circle-artifacts.com/gh/qorio/omni/61/artifacts/0/tmp/circle-artifacts.lteqSBx/linux_amd64/shorty?circle-token=b71701145614b93a382a8e3b5d633ee71c360315
mv shorty\?circle-token\=b71701145614b93a382a8e3b5d633ee71c360315 shorty
chmod a+x shorty
```

Make sure to copy the GeoIp file as well.

```
wget https://circle-artifacts.com/gh/qorio/omni/62/artifacts/0/tmp/circle-artifacts.7B2an1a/GeoLiteCity.dat?circle-token=b71701145614b93a382a8e3b5d633ee71c360315
mv GeoLiteCity.dat\?circle-token\=b71701145614b93a382a8e3b5d633ee71c360315 GeoLiteCity.dat
```

## Starting up Shorty
- Shorty uses unix domain sockets instead of tcp port
- Shorty requires one admin port
