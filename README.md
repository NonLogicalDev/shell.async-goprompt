# Not Your Average Async ZSH Shell Prompt

Inspired by: 
* https://github.com/nbari/slick
* https://github.com/ericfreese/zsh-efgit-prompt
* By the idea that Prompt Should NOT introduce any LAG
* By the pain of working in a gargantous monorepo where `git status` used to take over `10` seconds to run.

## Selling Points:

* Packs some punch, with lightning speed never seen before!
* This prompt is truly **INSTA**, as fast as no prompt at all.
* Zero lag between pressing enter and being able to type your next command.
* Truly and faithfully asynchroneous, can cope with most bloated Git monorepos out there without introducing lag. 
	* **#this_was_the_reason_this_prompt_was_created**
* The only prompt out there with support for **Stacked Git**.
	* **#another_reason**
* Nearly the only prompt out there to use ZLE File Descriptor co-routines effectively.

![Demo Of GoPrompt With ZLE](./assets/Kapture%202022-07-26%20at%2010.45.33.gif "Capture")

GoPrompt is lightning fast, and truly and faithfully asynchroneous prompt based on ZLE File Descriptor co-routines, with default implementation in a simple GoLang package.

## Install Today

I am periodically uploding new versions of pre-built binaries under `Releases`:

https://github.com/NonLogicalDev/shell.async-goprompt/releases/latest

Alternatively if you have `GoLang` installed you can install it directly from source:

```
go install github.com/NonLogicalDev/shell.async-goprompt/cmd/goprompt@latest
```

And if you want to build it yourself and/or contribute, feel free to checkout [#Install](#install) section below, for instruction of how to build it from source locally.

## Install Into Shell

```
$ goprompt install zsh >> ~/.zshrc
```

## Default Renderer supports:

### Example:

```
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
		* (`>`) - normal (command edit mode)
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

## Install

```
$ eval "$(gimme 1.20)"
$ make install USR_BIN_DIR="$HOME/bin"
$ goprompt install zsh >> ~/.zshrc
```
