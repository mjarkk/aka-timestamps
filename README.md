# Ask Kati Anything! Time stamps generator :^)

This generates timestamps for the [Ask Kati Anything!](https://www.youtube.com/playlist?list=PLMSjrqhPvOoZrz95tshKA9tIymbqxNxKn) podcast, it's a podcast filled with mental health questions and answers.
I'm not really interested in all questions so i usually search for the time stamps though sadly the makers of the podcast don't include them and for the community it's quit a bit of work so i made a small tool to automatically generate them.

# Setup:
1. Download the latests [youtube-dl from the releases page](https://github.com/ytdl-org/youtube-dl/releases) and place the file in the root of this project with the name: `./youtube-dl` *(You;ll need the latests version otherwhise the descriptions won't download fully)*
2. Make sure you have insatlled golang
3. Build the binary: `go build`

# Run:
```sh
./aka-timestamps "https://www.youtube.com/watch?v=7vFEkVQiF_c"
```

# Things that could be improved:
- Do not have to spesify the video to download
- Cache downloaded results
- Automaticlly detect common words like `a, i, the` instaid of having a pre defined blacklist
