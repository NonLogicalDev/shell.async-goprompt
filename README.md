# Async ZLE Based Prompt

(Implemented In GoLang)

## TLDR

This is a non-blocking asynchronous prompt based on ZLE File Descriptor Handlers.

The prompt query and rendering can be done via any command as long as it follows a line delimited protocol to communicate between the query and rendering components.


![Demo Of GoPrompt With ZLE](./assets/Kapture%202022-07-26%20at%2010.45.33.gif "Capture")

## Reference

You can find the ZSH/ZLE integration in:

* [prompt_asynczle_setup.zsh](./plugin/zsh/prompt_asynczle_setup.zsh)

And the main POC query/rendering logic is implemented in GO

* [goprompt](./cmd/goprompt)

## Install

```
$ eval "$(gimme 1.20)"
$ make install
$ goprompt install zsh.zshrc >> ~/.zshrc
```
