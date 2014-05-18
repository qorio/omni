# Server Configuraton

## Console

[DigitalOcean](https://cloud.digitalocean.com/droplets)

## DNS

[Name.com](https://www.name.com/account/domain/details/qor.io#dns)


## Software

List of sofware

Open Source

- Nginx (all)
- Redis (Shorty)
- Java (Stats)
- Logstash (Stats)
- ElasticSearch (Stats)

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

### Heartbleed (OpenSSL) Vulnerability

[Blog from DigitalOcean](https://www.digitalocean.com/community/articles/how-to-protect-your-server-against-the-heartbleed-openssl-vulnerability)

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
./install_server.sh
```

Some useful paths for redis:

```
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


### Java
On Ubuntu 12.10 - default is OpenJDK 7, but Oracle JDK is recommended.

```
apt-get install software-properties-common
apt-get install python-software-properties
add-apt-repository ppa:webupd8team/java
apt-get update
apt-get install oracle-java7-installer
```

Verify java version

```
root@stats1:~# java -version
java version "1.7.0_55"
Java(TM) SE Runtime Environment (build 1.7.0_55-b13)
Java HotSpot(TM) 64-Bit Server VM (build 24.55-b03, mixed mode)
```

### Logstash

[Documentation](http://logstash.net/docs/1.4.0/tutorials/getting-started-with-logstash)

```
curl -O https://download.elasticsearch.org/logstash/logstash/logstash-1.4.0.tar.gz
tar xvfz logstash-1.4.0.tar.gz
ln -s logstash-1.4.0 logstash
```

### ElasticSearch

#### Core ES service

Download Debian package from [elasticsearch.org/download](http://www.elasticsearch.org/download/)

[Instruction on setting up ElasticSearch as a service](http://www.elasticsearch.org/guide/en/elasticsearch/reference/current/setup-service.html)

```
wget https://download.elasticsearch.org/elasticsearch/elasticsearch/elasticsearch-1.1.1.deb
dpkg -i elasticsearch-1.1.1.deb
update-rc.d elasticsearch defaults 95 10
sudo /etc/init.d/elasticsearch start
```

Verify it's running

```
root@stats1:~# curl 'http://localhost:9200/_search?pretty'
{
  "took" : 3,
  "timed_out" : false,
  "_shards" : {
    "total" : 0,
    "successful" : 0,
    "failed" : 0
  },
  "hits" : {
    "total" : 0,
    "max_score" : 0.0,
    "hits" : [ ]
  }
}
```

Some files / directories to know:

- Installation home: `/usr/share/elasticsearch`
- PID file:  `/var/run/elasticsearch.pid`
- Log directory, defaults to `/var/log/elasticsearch`

Data directory

```
find /var/lib/elasticsearch/
/var/lib/elasticsearch/
/var/lib/elasticsearch/elasticsearch
/var/lib/elasticsearch/elasticsearch/nodes
/var/lib/elasticsearch/elasticsearch/nodes/0
/var/lib/elasticsearch/elasticsearch/nodes/0/_state
/var/lib/elasticsearch/elasticsearch/nodes/0/_state/global-1
/var/lib/elasticsearch/elasticsearch/nodes/0/node.lock
```

Config files
```
find /etc/elasticsearch/
/etc/elasticsearch/
/etc/elasticsearch/logging.yml
/etc/elasticsearch/elasticsearch.yml
```

#### Uitlities

[Monitor / Dashboard @ stats1.qor.io](http://stats1.qor.io:9200/_plugin/kopf/)

```
/usr/share/elasticsearch/bin/plugin -install lmenezes/elasticsearch-kopf
```

## Setting up Dashboard

Dashboard is running on the `web1` host.  See DNS: https://www.name.com/account/domain/details/qor.io#dns

### Files and Permissions

The files in `omni/www/kibana` maps to `/var/www/kibana` on host

On host:

```
chown -R www-data:www-data /var/www/kibana
chmod 755 /var/www
```

Change `config.js` to point to `https://stats1.qor.io`

### Nginx Setup

From `omni/etc/nginx` in local git repo:

```
scp web1.conf root@web1.qor.io:/etc/nginx/sites-available/console.qor.io
```
On host:
```
root@web1:/etc/nginx# find sites-available/
sites-available/
sites-available/default
sites-available/console.qor.io

root@web1:/etc/nginx# find sites-enabled/
sites-enabled/
sites-enabled/console.qor.io  # symlinked to file console.qor.io in sites-available
```

```
service nginx restart
```

The console is available at https://console.qor.io
