---
title: "Enforcing GitHub repository settings with Terraform"
summary: "How I manage my personal projects GitHub configuration"
date: "2025-01-23"
---

GitHub allows to define template repositories which content can be used as a base for new ones.

![template](/template.png)

I created such a template myself at https://github.com/juliendoutre/template. It contains a `CODEOWNER`, a `.gitignore`, a `README.md`, a `LICENSE`, and a `dependabot.yaml` files configured for ecosystems I use often.

Unfortunately, templating does not apply settings to a repository. In order to do this, I created https://github.com/juliendoutre/factory, a script that enables me to enforce settings on existing repositories, or create new ones with safe defaults.

The heavy lifting of the tool is done with Terraform which I like the declarative syntax and readability very much. I defined a few resources using the official [GitHub Terraform provider](https://registry.terraform.io/providers/integrations/github/latest/docs).

{{< alert "none" >}}
I use the `lifecycle { ignore_changes = [...] }` block to ignore some values that could be problematic. For instance, as I set the `template { ... }` block to point to my template repository, new repos created by the script are based on it, but older ones always show a diff in the Terraform plan.
{{</ alert >}}

The tool is then simply a bash script that dynamically import resources if they already exist into the Terraform state. It ultimately runs an apply that prints out the planned changes and asks the user if they wanna go ahead with them.

I'm kind of abusing the Terraform state management, as I'm using a local state file that I discard as soon as the script returns. But, this let's me see clearly what changes will be made by the script and without having to deal with any complex programmatic logic.

Feel free to try it based on other templates or with different settings and let me know what you think!

See you next time :wave:
