# A Tiny Zalo TTS wrapper
A small app written in [golang](https://golang.org) to simplify Zalo TTS integration.

These are the Docker Hub autobuild images located [here](https://hub.docker.com/r/locnh/tts/).

[![License](https://img.shields.io/github/license/locnh/tts)](/LICENSE)
[![Build Status](https://travis-ci.com/locnh/tts.svg?branch=master)](https://travis-ci.com/locnh/tts)
[![Docker Image Size (latest semver)](https://img.shields.io/docker/image-size/locnh/tts?sort=semver)](/Dockerfile)
[![Docker Image Version (latest semver)](https://img.shields.io/docker/v/locnh/tts?sort=semver)](/Dockerfile)
[![Docker](https://img.shields.io/docker/pulls/locnh/tts)](https://hub.docker.com/r/locnh/tts)

## Fearure

```bash
POST /raw -d 'Xin chào Việt Nam'

https://link-to-audio-file
```

```bash
POST /embeded -d 'Xin chào Việt Nam'

<audio controls autoplay><source src="https://link-to-audio-file" type="audio/mpeg"></audio>
```

## Usage
### Parameters
| Env Variable       | Mandatory | Default |
|--------------------|-----------|---------|
| ZALO_AI_API_KEY    | `yes`     | `null`  |
| ZALO_SPEAKER_ID    | `no`      | `1`     |
| ZALO_SPEAKER_SPEED | `no`      | `0.8`   |

More at [https://zalo.ai/docs/api/text-to-audio-converter](https://zalo.ai/docs/api/text-to-audio-converter)

### Run a Docker container

Default production mode

```sh
docker run -p 8080:8080 -e ZALO_AI_API_KEY=your-key -d locnh/tts
```

or add `-e GIN_MODE=debug` to debug

## Contribute
1. Fork me
2. Make changes
3. Create pull request
4. Grab a cup of tee and enjoy