# isotope-go

> [!IMPORTANT]
> If you're a hackclub terra reviewer, the demo link is the video. Unless you go to TJ, you cannot access or run a version of the bot yourself.

Isotope is a verification discord bot for TJ-related servers. It uses Ion's oauth2 flow to verify a user and makes them verified across multiple servers. It aims to replace the many individual verification bots that club servers have.

## Installation/Usage

### Running it

1. Clone the repo `git clone https://github.com/deR0R0/isotope-go`
2. Configure your stuff with the .env.example. **you have to be a TJ student to setup this bot.**
3. This project requires golang 1.27.0+. Install it.
4. Run it by first enabling C_GO for the database (`CGO_ENABLED=1`) and then running the bot (`go run .` or building it and running the binary)

### Guild setup
1. Run /guild settings and configure based on your needs

## How it works

It's currently a very simple bot; when users click on a verify button that's set up in the discord server, a login link will appear, which is generated from the oauth2 package from Google. After clicking "Authorize" in the oauth page of Ion, the state (associated with a user) is then used to associate a code with the user, the server does a code exchange for an oauth token and the token stuff is stored in the database.

I'm planning on making it so the bot doesn't have to store oauth tokens to still have verification work (by storing other stuff like Ion IDs or names).