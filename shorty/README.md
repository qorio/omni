
# Shorty

## Introduction

A url shortener and http redirector that understands different platforms and devices.  A single
url can render device-specific urls including custom url schemes such as `itms-service://` on
Apple devices for AppStore app, or `fb://` for Facebook app.

As of today (April 22, 2014), bit.ly does not appear to support this; neither does Google (goo.gl).

Some basic features:
- Platform-specific routing
    - Platform
    - OS
    - Browser
- Custom URL schemes allowed
- Stats
    - Cookie for unique visits
    - GeoIP information
        - Country (e.g. US)
	- Region (e.g. CA)
	- City
	- Postal Code
    - Additional information on inbound http requests
        - IP
        - Lat/Long

## Usage

### Create short link:

```
# longUrl is the fallback URL, required.
shell$ cat test.json | python -mjson.tool
{
    "longUrl": "http://cnn.com",
    "rules": [
        {
            "destination": "http://apple.com/mac",
            "platform": "Macintosh"
        },
        {
            "destination": "http://apple.com/iphone",
            "platform": "iPhone"
        },
        {
            "destination": "http://apple.com/ipod",
            "platform": "iPod*"
        },
        {
            "destination": "http://apple.com/ipad",
            "platform": "iPad"
        },
        {
            "destination": "http://android.com",
            "os": "Android*"
        }
    ]
}

# The url is http://qor.io/3US1O2fE
shell$ curl -X POST -s -d @test.json 'https://qor.io/api/v1/url' | python -mjson.tool
{
    "created": "2014-04-22T17:25:18.448305193Z",
    "destination": "http://cnn.com",
    "id": "3US1O2fE",
    "rules": [
        {
            "destination": "http://apple.com/mac",
            "platform": "Macintosh"
        },
        {
            "destination": "http://apple.com/iphone",
            "platform": "iPhone"
        },
        {
            "destination": "http://apple.com/ipod",
            "platform": "iPod*"
        },
        {
            "destination": "http://apple.com/ipad",
            "platform": "iPad"
        },
        {
            "destination": "http://android.com",
            "os": "Android*"
        }
    ]
}
```

### Getting stats:

```
shell$ curl -s 'https://qor.io/api/v1/stats/3US1O2fE' | python -mjson.tool
{
    "config": {
        "created": "2014-04-22T17:25:18.448305193Z",
        "destination": "http://cnn.com",
        "id": "3US1O2fE",
        "rules": [
            {
                "destination": "http://apple.com/mac",
                "platform": "Macintosh"
            },
            {
                "destination": "http://apple.com/iphone",
                "platform": "iPhone"
            },
            {
                "destination": "http://apple.com/ipod",
                "platform": "iPod*"
            },
            {
                "destination": "http://apple.com/ipad",
                "platform": "iPad"
            },
            {
                "destination": "http://android.com",
                "os": "Android*"
            }
        ]
    },
    "hits": 4,
    "id": "3US1O2fE",
    "summary": {
        "browsers": [
            {
                "name": "Chrome",
                "value": 2
            },
            {
                "name": "Safari",
                "value": 2
            }
        ],
        "cities": [
            {
                "name": "San Francisco",
                "value": 3
            },
            {
                "name": "Dixon",
                "value": 1
            }
        ],
        "countries": [
            {
                "name": "US",
                "value": 4
            }
        ],
        "os": [
            {
                "name": "CPU iPhone OS 7_1 like Mac OS X",
                "value": 2
            },
            {
                "name": "Android 4.3",
                "value": 1
            },
            {
                "name": "Intel Mac OS X 10_9_2",
                "value": 1
            }
        ],
        "platform": [
            {
                "name": "iPhone",
                "value": 1
            },
            {
                "name": "Linux",
                "value": 1
            },
            {
                "name": "iPod touch",
                "value": 1
            },
            {
                "name": "Macintosh",
                "value": 1
            }
        ],
        "postalcodes": [
            {
                "name": "94133",
                "value": 3
            },
            {
                "name": "95620",
                "value": 1
            }
        ],
        "referrers": [
            {
                "name": "DIRECT",
                "value": 4
            }
        ],
        "regions": [
            {
                "name": "CA",
                "value": 4
            }
        ]
    },
    "uniques": 4,
    "when": "4 minutes ago"
}
```
