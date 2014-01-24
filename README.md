mementor
========

Simple tool to display messages from a stored list.

Use cases:

* display reminders about stuff everytime time you start the console
* display vim tips whenever you start vim
* create a git hook to display useful tips about those fancy commands you keep forgetting


Installation
============

Simple install
--------------
The program comes with batteries included. Simply copy `bin/mementor` (or create a symlink) to your $PATH.

Installing from source
-------------------
Alternatively, you can compile from source.

[Install go](http://golang.org/doc/install) if you haven't done so yet. Thereafter:

```bash
go build
```

Copy or symlink the compiled binary `mementor` into your $PATH.

Usage
=====

```bash
mementor fetch
```

Fetches a single message from the stored list at random. This is the default action, so `mementor` will fetch a message.


```bash
mementor list
```

Displays the list of stored messages.

```bash
mementor add "Velit esse molestie consequat, vel illum dolore eu feugiat nulla"
```

Adds new message to the stored list.

```bash
mementor rm 2
```

Removes message #2 from the stored list.

```bash
mementor help
```

Does what you think it does.


Multiple message lists are supported. The default message lists is stored at $HOME/.mementor/mementos.json but you may choose other locations with the *-f* option:

```bash
mementor -f ~/vim-tips.txt list
```

Use cases
=========

.bashrc
-------
Put this into your `.bashrc`:

```
mementor fetch
```

.vimrc
------
Put this into your `.vimrc`:

```
let useful_tip=system('mementor fetch')
echo useful_tip
```

