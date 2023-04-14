---
title: "How I built this website"
description: "An explanation of how this website is built and served"
date: "2023-04-03"
---
I wanted to have my own blog for a long time. Finally, some day last week, I found the time to set up one. This article will reflect on my thought process and the stack I ended up choosing.

My requirements were pretty straigthforward. I wanted:
* **to write posts using just markdown.** I'm not a fan of fancy text editors, they're not reactive enough. Being able to write in my IDE ([vscode](https://code.visualstudio.com/)) where I already manage my code and my terminal makes it easier for me to gather all my tools at the same place.
* **to host the website myself**, not just posting on [Medium](https://medium.com/) or [Wordpress](https://wordpress.com/) for instance. I wanted to be able to control the very content I would expose publically from the articles' body to the pages' HTML.
* **the solution to be as cheap as possible.** Do I really need to elaborate?

Additionally, I did not want to build something from scratch, especially not relying on JS frameworks like [React](https://react.dev/) which are way too overkill for small projects. Moreover I'm not a frontend engineer, I'm really bad at drawing with CSS (I still struggle centering `<div>` inside flex boxes, how do these guys have their tags behave as they want them to?! :exploding_head:).

Naturally, the go-to for static websites supporting markdown is [Hugo](https://gohugo.io/).
It did not take me a lot of time to realize it was a perfect fit for my use case:

```shell
brew install hugo
hugo new site juliendoutre.github.io
```

and I had my project initialized.

```shell
hugo serve
```

and I had a locally version of the website running.

What took me the most time to figure out was obviously (and as foresaw my friend Edouard) which theme to use.

I had vague memories of the [Gitbook project](https://www.gitbook.com/) which I liked the chapter-based organization with a nice vertical left side panel. I checked their website and noticed they stopped supporting their self-hosted solution to focus on a SAAS platform. Too bad :shrug: Plus it did not really fit a blog post use case, missing a lot of features I could find elsewhere.

After exploring the Hugo theme showcase I ended up choosing the [Congo theme](https://jpanther.github.io/congo/) which was appealing to me for the following reasons:
* it has a search bar
* it has a dark mode
* the home page can show as a profile page
* configuration options are crystal clear
* it uses [Tailwind](https://tailwindui.com/) and I heard it's the cool new thing.

> The only non-trivial customization was changing the default flaticon. I had to lookup some other persons' blogs source code to find the right file names to use in the `static` folder of my project.

Regarding the hosting, I considered for a second deploying everything in my [AWS](https://aws.amazon.com/) account using [S3](https://aws.amazon.com/s3/) and [CloudFront](https://aws.amazon.com/cloudfront/). But in the end, [GitHub pages](https://pages.github.com/) is easier to use, integrates seamlessly with [GitHub actions](https://github.com/features/actions), plus it's free (I don't even have to pay for a domain name)!

My current development workflow is pushing to the `main` branch, waiting for the GitHub action to generate the `public` folder to the `gh-pages` branch, which is then served by GitHub pages on the domain matching the repository's name. I'll see how this can be improved in the future!

In the meantime, if you're curious, you can check this website source code at https://github.com/juliendoutre/juliendoutre.github.io :nerd_face:

See you next time :wave:
