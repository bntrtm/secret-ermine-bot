# Secret Ermine Bot

## About
The Secret Ermine bot for [Stoat](https://stoat.chat) provides users with a Secret Santa service used to manage gift exchanges between users.

## Background
I have long attended a TTRPG campaign wherein some friends and I used Discord to communicate. When moving our communications to Stoat, I checked for any Stoat-based bots providing a Secret Santa service, which we would make use of each year when the holidays rolled around. Finding none, I decided to try my hand at writing a bot for Stoat, and this is the result!

## Features
- Reaction-based participant sign-up
- Manage multiple Secret Santa events per bot instance (one per server)
- Write anonymized messages to your Secret Santa or Giftee
- Easier command syntax under DM context
  - In server channels: `@BOT-HANDLE !<command>`
  - In bot DMs, mention becomes optional: `!<command>`

# Installation

## Notice

There may or may not be instances of this bot already running that are discoverable via the Stoat platform. If you find that to be the case, and are satisfied with the way such an instance is configured, it should have no trouble managing your events as well!

Nonetheless, installing and running the bot yourself is always an option, especially if you prefer to have control over the bot instance you wish to use.

## Install with Go

With Go installed:

```bash
go install github.com/bntrtm/secret-ermine
```

Then run it with `secret-ermine`.

Alternatively, you can build from source after cloning with `git`:

```
git clone github.com/bntrtm/secret-ermine.git
cd secret-ermine
go build
```

## Quick Start

First things first, you'll need a Stoat bot set up. You can create a new bot via the Stoat platform and copy a token to use from there:

```
My account -> My bots -> Create/Select Bot -> Copy ID
```

For the bot to use the program, you need the `BOT_TOKEN` environment variable set. You can set it within a `.env` file under the same directory as the binary:

```bash
BOT_TOKEN=your-stoat-bot-token-here
```

With the environment variable set, you can now run the program!

```bash
./secret-ermine
```

You should see terminal output indicating an attempt to make a WebSocket connection. You will know that the bot is ready for action when it lets you know that the connection was resolved.

To add the bot to a Stoat server, use the `Invite Bot` tool, also found on the webpage provided after selecting the bot.

# Usage

## Permissions
Ensure that the bot has permissions to read, write, and send messages, as well as masquerade permissions, so that it can function as expected.

## Commands
To command the bot, start a message with one of the following prefixes, dependent on context:

1) `@BOT-HANDLE !<command>`
2) `!<command>`

The first may be used in either server channels OR direct messages with the bot. The second may  be used only in direct messages with the bot.

To get a list of commands you can run, use the `help` command within a server channel or direct message to the bot:

```
@BOT-HANDLE !help
```

The bot will respond with a neat list of commands and instructions for each!

## How It Works

A Secret Santa event progresses as follows:

1) An organizer uses the `new` command to start a new Secret Santa event with the bot.
2) The bot sends a message to the same channel.
3) Users apply one or more emoji reactions to the join message.
4) With three or more unique participants having reacted, the organizer can use the `start` command to begin the event. They may also `cancel` at any time, as well.
5) Between the start date and distribution date, users can send anonymized messages to their Secret Santas or giftees, using the bot as a mediator.
6) When the distribution date comes, gifts are exchanged!

# Contribution

I'm happy to discuss whatever contribution you're interested in exploring for the project!
The rules are few:

1) Ask first before writing a pull request that may get rejected!
2) No AI-generated code
3) Adhere to `go fmt` styling
4) Include unit tests where possible

# Attribution

See the [attribution file](ATTRIBUTION.md) for full details.
