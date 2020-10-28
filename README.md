# Ask Kati Anything! Time stamps generator :^)

This generates timestamps for the [Ask Kati Anything!](https://www.youtube.com/playlist?list=PLMSjrqhPvOoZrz95tshKA9tIymbqxNxKn) podcast, it's a podcast filled with mental health questions and answers.
I'm not really interested in all questions so i usually search for the time stamps though sadly the makers of the podcast don't include them and for the community it's quit a bit of work so i made a small tool to automatically generate them.

## Setup
```sh
# Build container
docker build -t aka-timestamps:latest .

# Generate key(s)
# From: https://www.howtogeek.com/howto/30184/10-ways-to-generate-a-random-password-from-the-command-line/
tr -cd '[:alnum:]' < /dev/urandom | fold -w30 | head -n1

# Run the container
docker run -d --env "AKA_KEYS=VERY_SECRET_KEY_HERE,OPTIONAL_SECOND_KEY,ANOTHER" -p 127.0.0.1:9090:9090 --name aka-timestamps --restart always aka-timestamps:latest
```
*For ssl use a reverse proxy. The webapp is setup using (Vercel)[https://vercel.com/]*

