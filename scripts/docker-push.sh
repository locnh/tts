#!/bin/bash
echo "$DOCKER_PASSWORD" | docker login --username "$DOCKER_USERNAME" --password-stdin

if [ $TRAVIS_BRANCH != "master" ]; then
    docker tag locnh/tts locnh/tts:$TRAVIS_BRANCH
    docker push locnh/tts:$TRAVIS_BRANCH
else
    docker push locnh/tts
fi
