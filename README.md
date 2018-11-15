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
* /quote add *genius quote* - Store *genius quote* for later.

Admin commands (assuming I can tell if the user is an admin or not):

* /quote list - List all known quotes.
* /quote delete *x* - Delete quote number *x*.

Periodically post a random quote to a specified channel. Default is every 24
hours in `~town-square` if there's activity there.

TODO: Monitor multiple channels.

Output:

> I feel pretty. -- @shane
> It's like _Speed_ but more stupid. -- @chris

Quotebot could also handle responses, with a set of commands like this:

* /response add "*trigger*" "*response text*" - Say *response text* when
  someone writes *trigger*.

Admin commands for `/response`:

* /response list - List all known responses.
* /response delete "*trigger*" - Delete the response for *trigger*.

The responses would have their own channel separate from the quotes, and a 1/*x*
chance of responding to a matched trigger so it doesn't get annoying.

TODO: Monitor multiple channels.

## Credits

* [Mattermost's plugin sample](https://github.com/mattermost/mattermost-plugin-sample)
* Icon from [Nick Roach](http://www.elegantthemes.com/)'s GPL'd
  [Circle Icons](https://www.iconfinder.com/iconsets/circle-icons-1) set.
