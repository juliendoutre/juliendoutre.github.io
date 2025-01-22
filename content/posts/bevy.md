---
title: "I (almost) wrote a game in Rust with Bevy!"
summary: "But in the end I just messed around with GitHub actions"
date: "2024-05-11"
---

I always wanted to create a video game. It's a topic I approached a bit while working on some projects like [a solution to the Synacor challenge](https://github.com/juliendoutre/synacor-challenge) or a [R8 emulator implementation](https://github.com/juliendoutre/r8). But there were mostly coding challenges and did not involve any actual game design.

I played a bit with [Unity](https://unity.com/) and even created a [small platformer game](https://github.com/GravityJump/GravityJump) with some friends for a school project ([play it online here!](https://gravityjump.github.io/GravityJump/)). However I was confused by the code organization. I really wanted to manage the whole app as code and could not really achieve this. Everything had to be managed through scripts attached to objects only tracked in the IDE or weird metadata files. This was not satisfying to my nascent software engineering mindset!

A couple of years ago, I stumbled upon the [Amethyst project](https://github.com/amethyst/amethyst). I was really excited about the ECS approach and read some guides but never came to create anything concrete. And then the project stopped :cry:

However some developpers seemed decided to continue the adventure and started [Bevy](https://bevyengine.org/). It's really close to Amethyst concept-wise but they seemed to get rid of some of the cumbersome type declarations and only kept the best from this framework.

For those not familiar with ECS, it stands for Entity-Component-Systems. It's a way to design games based on entities which can get attached components and updated by systems which updates their state.

I decided to give it a try and started a new GitHub repository: https://github.com/juliendoutre/froggy.

I took inspiration from https://github.com/bevyengine/bevy/blob/latest/examples/games/game_menu.rs to create a simple splash screen and a game menu with two buttons.

Assets come from https://www.kenney.nl which provides an amazing collection of game assets for free :heart:

Designing UI components on a canvas was similar as writing HTML nodes but in Rust... which felt rather cumbersome. Once I had a satisfying rendering, I decided to release my game. I skimmed through https://bevy-cheatbook.github.io/platforms.html and noticed Bevy support WASM!

As for the previous game I was working for, I decided to release it on GitHub pages (free hosting for the win) but this time I wanted to automate this a bit. And it happens the Rust toolchain is pretty well integrated in the GitHub actions ecosystem.

My first step was to add a CI workflow with several jobs to check code's formating, run clippy, tests, build the project, and check the lock file is up to date.

```yaml
name: CI

# The workflow should only run for commits in PRs and the main branch.
on:
  push:
    branches:
      - main
  pull_request:
    branches:
      - '*'

# Let's use concurrency groups to cancel stale jobs except on the main branch.
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: ${{ github.ref != 'main' }}

# Always explicitly set a workflow permissions!
permissions:
  contents: read
```

One optimization I used is to cache the `.target` and `.cargo` folders so that they can be used across jobs. All my jobs therefore start with the following steps:
```yaml
jobs:
  my-job:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/cache@v4
      with:
          path: |
            ~/.cargo/bin/
            ~/.cargo/registry/index/
            ~/.cargo/registry/cache/
            ~/.cargo/git/db/
            target/
          key: ${{ runner.os }}-cargo-${{ hashFiles('**/Cargo.lock') }}
      - run: rustup update stable && rustup default stable
```

I noticed compilation errors at build time because of missing dev libraries that I was able to fix simply with:
```yaml
- run: sudo apt-get update && sudo apt-get install -y libasound2-dev libudev-dev
```

Then I created a CD workflow to build and deploy the game to a GitHub page:

```yaml
name: CD

# The workflow should only run for commits on the main branch.
on:
  push:
    branches:
      - main

# Always explicitly set a workflow permissions!
permissions:
  contents: read
```

with the following jobs:

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/cache@v4
        with:
          path: |
            ~/.cargo/bin/
            ~/.cargo/registry/index/
            ~/.cargo/registry/cache/
            ~/.cargo/git/db/
            target/
          key: ${{ runner.os }}-cargo-${{ hashFiles('**/Cargo.lock') }}
      - run: rustup update stable && rustup default stable
      # We need to make sure the Rust toolchain supports WASM.
      - run: rustup target install wasm32-unknown-unknown
      # Building a WASM binary.
      - run: cargo build --release --target wasm32-unknown-unknown
      # Installing some tools to optimizie the WASM binary.
      - run: cargo install wasm-bindgen-cli@0.2.92 wasm-opt@0.116.1
      # Generating some JS code to load the WASM in a HTML canvas.
      - run: wasm-bindgen --no-typescript --target web --out-dir ./build/ --out-name froggy ./target/wasm32-unknown-unknown/release/froggy.wasm
      # Optimizing the binary for size. Experimentally, it decreased the size by 2 which saves some bandwidth for the website users (from 30M to 15M).
      - run: wasm-opt ./build/froggy_bg.wasm -o ./build/froggy_bg.wasm -Oz
      # Adding a dead simple HTML file to load the JS code and define the aforementioned canvas.
      - run: cp ./www/index.html ./build/index.html
      # Copying the assets into the build folder so that they are bundled too and served by the website.
      - run: cp -r ./assets ./build/assets
      # Uploading the build folder to artifacts.
      - uses: actions/configure-pages@v5
      - uses: actions/upload-pages-artifact@v3
        with:
          path: ./build
  deploy:
    # GitHub pages are now action based and not simply based on a Git branch.
    permissions:
      pages: write
      id-token: write
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    needs: build
    steps:
      - uses: actions/deploy-pages@v4
        id: deployment
```

It takes about 9 minutes to build and deploy the website. And here is the final result: https://juliendoutre.github.io/froggy!

This was a nice journey but I noticed some caveats:
- bevy does not support hot reloading
- writing Rust code does not let any room for quick hacks which is often needed when developing a game.

In the end, most of the points mentioned in https://loglog.games/blog/leaving-rust-gamedev.

So next, I'd like to try [Godot](https://godotengine.org/) and give another chance to more "classic" game engines.

See you next time :wave:
