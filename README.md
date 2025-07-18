# Not Your Average Async ZSH/FISH Shell Prompt

<center>

![Project Logo](./assets/logo_w1000.png)

</center>

Inspired by:
* https://github.com/nbari/slick
* https://github.com/ericfreese/zsh-efgit-prompt
* https://github.com/acomagu/fish-async-prompt/blob/master/conf.d/__async_prompt.fish
* https://github.com/jorgebucaran/hydro/blob/main/conf.d/hydro.fish

* By the idea that Prompt Should NOT introduce any LAG
* By the pain of working in a gargantous monorepo where `git status` used to take over `10` seconds to run.


## Selling Points:

* Packs some punch, with lightning speed never seen before!
* This prompt is truly **⚡️ INSTA ⚡️**, as fast as no prompt at all.
* Zero lag between pressing enter and being able to type your next command.
* Truly and faithfully asynchronous, can cope with most bloated Git monorepos out there without introducing lag.
	* This was the first and foremost requirement
* Pretty much the only prompt out there with native support for **VCS: Stacked Git** and **VCS: Sappling**.
* Pretty much the only prompt out there to use ZLE File Descriptor for async work.

![Demo Of GoPrompt With ZLE (ZSH)](./assets/Kapture%202022-07-26%20at%2010.45.33.gif "Capture")

GoPrompt is lightning fast, and truly and faithfully asynchronous prompt based on ZLE File Descriptor co-routines, with default theme/query implementation in a very simple to extend GoLang package.

## Quick Install

The latest releases are available under:

https://github.com/NonLogicalDev/shell.async-goprompt/releases/latest

Install latest using:

```sh
curl -sfL https://raw.githubusercontent.com/NonLogicalDev/shell.async-goprompt/main/install.sh | bash -
```

This will install `goprompt` under `~/.local/bin`. Please ensure that it is in your `$PATH`.

Alternatively if you have `GoLang` installed you can install it directly from source:

```
go install github.com/NonLogicalDev/shell.async-goprompt/cmd/goprompt@latest
```

And if you want to build it yourself and/or contribute, feel free to checkout [#Build Instructions](#build-instructions) section below, for guidance on how to build it from source locally.

## Install Into Shell (ZSH)

Try for one session:
```sh
$ eval "$(goprompt install zsh)"
```

Install permanently:
```sh
$ goprompt install zsh >> ~/.zshrc
```

## Install Into Shell (FISH)

Try for one session:
```sh
$ eval "$(goprompt install fish)"
```

Install permanently:
```sh
$ goprompt install fish >> ~/.config/fish/conf.d/50-goprompt.fish
```

## Default Renderer supports:

### Example:

```sh
# After running this:

	$ ( sleep 570; exit 130 )

# Example prompt with most integrations displayed:

	:: {git:main:&:[+1:-0]} {stg:readme:1/2}
	:: [130] (vifm) (~/U/P/shell.async-goprompt) 9m30s [22:18:42 02/20/23]
	>

# After normal (faster, errorless) execution:

	:: {git:main:&:[+1:-0]} {stg:readme:1/2}
	:: (vifm) (~/U/P/shell.async-goprompt) [22:18:42 02/20/23]
	>

# When outside of VCS root:

	:: ------------------------------
	:: (vifm) (~/U/Projects) [22:18:42 02/20/23]
	>

```

### Features:

* Ascii-only but still pretty
	* Because there are so many bad terminal emulators out there.

* `Pure`-like:
	* Truncated Current Path Display (`~/U/P/shell.async-goprompt`)
	* last command duration (`9m30s`)
	* last command exit status (`[130]`)
		* makes debugging shell scripts and `test` commands that much easier
	* Vim Mode indicator support
		* (`>`) - default (insert mode)
		* (`<`) - normal (command edit mode)
	* SSH / Remote process detection

* Prompt Query State:
	* (`:?`) Prompt Query Ongoing
	* (`::`) Prompt Query Finished
	* (`:x`) Prompt Query Timeout or Failed

* Current Date Display (`[22:00:18 02/20/23]`)
* Parent Process name (to see when you are in a nested session like in VIFM) (`(vifm)`)

* VCS: Git (`{git:main:&:[+1:-0]}`)
	* **[works fast even in a gigantic sluggish monorepo]**
	* Current Branch (`main`)
	* Index/Worktree Dirty Status (`&`)
	* Rebase Detection (`:rebase`)
	* Lag Behind Remote (`[+1:-0]`)
		* Number of unpublished commits (`+1`)
		* Number of new remote commits (`-0`)

* VCS: Git+Stacked Git (`{stg:readme:1/2}`)
	* Current Patch (`readme`)
	* Patch stack size and location in the stack (`1/2`)
	* Metadata out of sync alert (`stg` badge will turn red)

* VCS: Sappling (new VCS from Facebook) (`{spl:feature1:&}`)
	* Current Active Bookmark (`feature1`)
	* Worktree Dirty status (`&`)

## Technology / Implementation Details

This is a non-blocking asynchronous prompt based on ZLE File Descriptor Handlers.

The prompt query and rendering can be done via any command as long as it follows a line delimited protocol to communicate between the query and rendering components.

### Protocol

The `query` command output must adhere to a line protocol to be effective in making use of ZLE File Descriptor handler.

The protocol used in my Async ZLE Implementation is dead simple and easy to implement and use. (So easy it initially was fully implemented in a ZSH script)

First of all the protocol is **new line** and **tab** delimited, where each line takes the following form:
```
$KEY1<tab char>$VALUE1
$KEY2<tab char>$VALUE2
```

This makes key parsing dead simple and allows values to be as complex as desired (so long as they dont contain new lines), for example they can be single line encoded JSON values.

Each empty line triggers a prompt refresh, kind of like a `sync` singal.
```
$KEY1<tab char>$VALUE1
$KEY2<tab char>$VALUE2
<empty>
$KEY3<tab char>$VALUE3
$KEY4<tab char>$VALUE4
```


This allows the prompt to periodically communicate that batch of data is ready. Being judicious in not sending `sync` signals leads to less visual jitter when prompt re-renders, in response to updated data.

Upon every `sync` signal renderer gets a newline concatentated list of Key Value Lines on its `STDIN`, and produces the actual prompt.

### Renderer

The only protocol on the renderer is that Renderer is expected to produce ZSH formatted prompt string base on newline delimited list of key values.

Overall the renderer is a bit like a pure `React` component `render` function.

## Reference

You can find the ZSH/ZLE integration in:

* [prompt_asynczle_setup.zsh](./plugin/zsh/prompt_asynczle_setup.zsh)

And the main query/rendering logic is implemented in GO

* [goprompt](./cmd/goprompt)

## Build Instructions

**Preferred Prerequisites:**

* [Mise](https://github.com/mise-tea/mise)
	* Used to manage GoLang dependencies
	* But you can install GoLang dependencies manually if you want

```sh
$ mise trust
$ make install USR_BIN_DIR="$HOME/bin"
$ goprompt install zsh >> ~/.zshrc
# or
$ goprompt install fish >> ~/.config/fish/conf.d/50-goprompt.fish
```
