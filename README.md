# ca.taffer.mm-quotebot

A Mattermost quotation bot.

![Chat icon](if_chat_1055095.png)

Inspired by Slack's Loading Messages and Slackbot.

I still don't know Go (see [Rolly](https://github.com/Taffer/ca.taffer.mm-rolly)
for more evidence!), so buckle up...

## Quotations

Quotebot remembers quotes you tell it about, and spits them out again when you
ask it to.

Commands:

* /quote - Regurgitate a random quote.
* /quote *x* - Show quote number *x*.
* /quote add *genius quote* - Store *genius quote* for later. Don't forget to
  include an attribution!
* /quote help - Show the help.

Admin commands:

* **TODO** /quote channel *x* - Monitor channel *x* for activity and randomly
  show quotes there.
* /quote delete *x* - Delete quote number *x*.
* **TODO** /quote interval *x* - The time between automatically posting quotes
  in a channel.
* /quote list - List all known quotes.

Periodically posts a random quote to a specified channel. Default is every 60
minutes in `~town-square` if there's activity there.

**TODO:** Make it periodically post.
**TODO:** Monitor multiple channels.

Output:

> I feel pretty. -- @shane

**TODO:** Handle responses, or make a Responsebot:

Quotebot _could_ also handle responses, with a set of commands like this:

* /response add "*trigger*" "*response text*" - Say *response text* when
  someone writes *trigger*.

Admin commands for `/response`:

* /response list - List all known responses.
* /response delete "*trigger*" - Delete the response for *trigger*.

The responses would have their own channel separate from the quotes, and a 1/*x*
chance of responding to a matched trigger so it doesn't get annoying.

> It's like _Speed_ but more stupid. -- @chris

## Credits

* [Mattermost's plugin sample](https://github.com/mattermost/mattermost-plugin-sample)
* Icon from [Nick Roach](http://www.elegantthemes.com/)'s GPL'd
  [Circle Icons](https://www.iconfinder.com/iconsets/circle-icons-1) set.
