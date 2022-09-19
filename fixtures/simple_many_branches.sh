#!/usr/bin/env bash

git init -b main
git commit --allow-empty -m "empty commit"
git branch foo
git branch bar
git branch baz

touch a
git add a
git commit -m "Add a"

git branch qux
git branch quux
