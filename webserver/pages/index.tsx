import Head from 'next/head'
import { useEffect, useState } from 'react'

const url = (path: string) => (window.location.host == 'localhost:3000' ? 'http://localhost:9090' : 'https://api.aka-podcast.mkopenga.com') + path

interface EP {
  number: number
  rawNumber: string
  name: string
  foundDescription: boolean
  foundVTT: boolean
  foundResults?: AnylizeResults
}

interface AnylizeResults {
  questions: QuestionType[]
  timeStamp: DetectedTimeStamp[]
  err: string
}

interface QuestionType {
  full: string
  searchable: string
  shortent: string
}

interface DetectedTimeStamp {
  questionIdx: number
  atStr: string
  found: boolean
}

interface Theme {
  defaultBg: string
  secondBgAndText: string
  importantText: string
  secondBgText: string
}

const defaultTheme: Theme = {
  defaultBg: '#eee',
  secondBgAndText: '#bbb',
  importantText: '#bbb',
  secondBgText: '#ccc',
}

const themes: Theme[] = [
  {
    defaultBg: '#f8a8a3',
    secondBgAndText: '#5fce7b',
    importantText: '#36b355',
    secondBgText: 'black',
  },
  {
    defaultBg: '#dee7e6',
    secondBgAndText: '#fb948d',
    importantText: '#d67e78',
    secondBgText: '#3c7440',
  },
  {
    defaultBg: '#df84b9',
    secondBgAndText: '#f1f9a0',
    importantText: '#c8cf79',
    secondBgText: '#1e0ed1',
  },
  {
    defaultBg: '#d8f4d7',
    secondBgAndText: '#4cd46e',
    importantText: '#2db64f',
    secondBgText: 'black',
  },
  {
    defaultBg: '#3402cb',
    secondBgAndText: '#2cfa29',
    importantText: '#a0ff9f',
    secondBgText: 'black',
  },
  {
    defaultBg: '#f8c3db',
    secondBgAndText: '#1404ce',
    importantText: '#0f01a8',
    secondBgText: '#fb8c58',
  },
  {
    defaultBg: '#fefcc7',
    secondBgAndText: '#53e25c',
    importantText: '#47c04f',
    secondBgText: '#400ae2',
  },
  {
    defaultBg: '#3a02f2',
    secondBgAndText: '#82f1e4',
    importantText: '#c1fdf5',
    secondBgText: '#000000',
  },
  {
    defaultBg: '#fafcc6',
    secondBgAndText: '#e3b4af',
    importantText: '#bb918c',
    secondBgText: '#5d07fe',
  },
  {
    defaultBg: '#eefc97',
    secondBgAndText: '#fa8d74',
    importantText: '#c56853',
    secondBgText: '#3a02f2',
  }
]

export default function Home() {
  const [eps, setEps] = useState(undefined as EP[] | undefined)
  const [key, setKey] = useState('')
  const [showKeyInput, setShowKeyInput] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [theme, setTheme] = useState(defaultTheme)
  const [transitions, setTransitions] = useState(false)

  const loadEps = async () => {
    try {
      const r = await fetch(url('/eps'))
      const eps = await r.json()
      setEps(eps)
    } catch (e) {
      setEps([])
    }
  }

  const checkNewVideos = async () => {
    try {
      setLoading(true)
      setError('')
      localStorage.setItem('aka-timestamps-key', key)
      const options = {
        method: "POST",
        body: JSON.stringify({ key })
      }
      const r = await fetch(url('/eps/re-fetch'), options)
      const data = r.json() as any
      if (data.error) {
        setError(data.error)
      } else {
        await new Promise(res => setTimeout(res, 500))
        await loadEps()
        setError('')
        setShowKeyInput(false)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    setTheme(themes[Math.floor(Math.random() * themes.length)])
    loadEps()

    const key = localStorage.getItem('aka-timestamps-key')
    if (key) setKey(key)

    setTimeout(() => setTransitions(true), 100);
  }, [])

  return (
    <>
      <Head>
        <title>AKA Timestamps</title>
        <link rel="icon" href="/favicon.ico" />
        <link href="https://fonts.googleapis.com/css2?family=Source+Serif+Pro&display=swap" rel="stylesheet" />
      </Head>

      <main>
        <div>
          <h1>AKA Timestamps</h1>
          <p className="info">This generates timestamps for the <a href="https://www.youtube.com/playlist?list=PLMSjrqhPvOoZrz95tshKA9tIymbqxNxKn">Ask Kati Anything! (AKA)</a> podcast, it's a podcast filled with mental health questions and answers.</p>
          {showKeyInput ?
            <div>
              <div>
                <p className="info">Because of Youtube's query limitations for videos i have limited the capability query to a handful of people, to get access to this please contact me at <a>mkopenga@gmail.com</a></p>
                <input autoFocus placeholder="Refresh key" value={key} onChange={e => setKey(e.target.value)} disabled={loading} />
                <button onClick={checkNewVideos} disabled={loading}>Check</button>
              </div>
              <div className="refresh-info">
                {loading ? <p>Checking for new videos..</p> : ''}
                {error ? <p className="error">{error}</p> : ''}
              </div>
            </div>
            :
            <div>
              <button onClick={() => setShowKeyInput(true)}>Check for new videos</button>
            </div>
          }

          {eps ? eps.map((ep, key) => <EpBlock theme={theme} key={key} ep={ep} />) : <EpBlock theme={theme} />}
          <p className="info"><a href="https://github.com/mjarkk/aka-timestamps">The code of this tool is open source and can be found here</a></p>
        </div>
      </main>
      <style jsx global>{`
        * {
          padding: 0px;
          margin: 0px;
          transition: ${transitions ? `background-color 0.2s, color 0.2s` : ``};
        }
        body {
          font-size: 18px;
          font-family: sans-serif;
          background-color: ${theme.defaultBg};
          color: ${theme.secondBgAndText};
        }
        h1, h2, h3 {
          font-family: 'Source Serif Pro', serif;
        }
        h1 {
          font-size: 2.6rem;
        }
        h3 {
          font-size: 1.4rem;
        }
        button {
          background-color: ${theme.secondBgAndText};
          color: ${theme.defaultBg};
          font-weight: bold;
          padding: 5px 15px;
          font-size: 1rem;
          border-radius: 10px;
          border: 0;
          cursor: pointer;
        }
        button:hover {
          background-color: ${theme.secondBgText};
        }
        input {
          padding: 5px 15px;
          background-color: ${theme.defaultBg};
          font-weight: bold;
          color: ${theme.secondBgAndText};
          font-size: 1rem;
          border: 0;
          border-bottom: 2px solid ${theme.secondBgAndText};
          margin-right: 10px;
        }
        input::-webkit-input-placeholder {
          color: ${theme.secondBgAndText};
          opacity: 0.5;
        }
        input:disabled {
          opacity: 0.5;
        }
        button:disabled {
          opacity: 0.5;
        }
      `}</style>
      <style jsx>{`
        main {
          max-width: 500px;
          margin: 0 auto;
          padding: 20px;
        }
        h1 {
          display: inline-block;
          margin: 40px 0px;
        }
        .info {
          margin-bottom: 25px;
          font-weight: bold;
        }
        .info a {
          color: ${theme.secondBgAndText};
        }
        .info:hover a {
          color: ${theme.importantText};
        }
        .refresh-info {
          padding: 10px 5px 0 5px;
        }
        .error {
          color: red;
        }
      `}</style>
    </>
  )
}

function EpBlock({ ep, theme }: { ep?: EP, theme: Theme }) {
  return (
    <div className="ep-block">
      <h3>{ep?.name}</h3>
      {ep?.foundResults?.questions && ep?.foundResults?.timeStamp
        ? <Timestamps res={ep.foundResults} theme={theme} />
        : <div className="meta">
          <p>{
            !ep ? `loading..`
              : !ep.foundDescription ? `Unable to read description of episode`
                : !ep.foundVTT ? `Unable to read description of episode`
                  : ep.foundResults?.err ? `Oh wired error: ${ep.foundResults?.err}`
                    : `Unable to get timestamps for this episode`
          }</p>
        </div>
      }
      <style jsx>{`
        .ep-block {
          background-color: ${theme.secondBgAndText};
          padding: 20px;
          margin: 40px 0px;
          border-radius: 10px;
          color: ${theme.secondBgText};
          overflow: hidden;
        }
        .ep-block h3 {
          margin-bottom: 10px;
        }
      `}</style>
    </div>
  )
}

function Timestamps({ res, theme }: { res: AnylizeResults, theme: Theme }) {
  const [timestamps, setTimestamps] = useState([] as DetectedTimeStamp[])

  useEffect(() => {
    setTimestamps(res.timeStamp.reduce((acc, item) => {
      if (acc.length == 0) {
        return [item]
      }
      if (acc[acc.length - 1].questionIdx != item.questionIdx) {
        acc.push(item)
      } else {
        acc[acc.length - 1] = item
      }
      return acc
    }, [] as DetectedTimeStamp[]))
  }, [res])

  const question = (idx: number) => res.questions[idx]

  const copy = async () => {
    const toCopy = timestamps.map(timestamp => `${timestamp.atStr} ${question(timestamp.questionIdx).shortent}`).join('\n')
    await navigator.clipboard.writeText(toCopy)
  }

  return (
    <div className="time-stamps">
      {timestamps.map((timestamp, key) =>
        <p key={key}>{timestamp.atStr} {question(timestamp.questionIdx).shortent}</p>
      )}
      <div>
        <button onClick={copy}>Copy timestamps</button>
      </div>
      <style jsx>{`
        p {
          font-size: 0.9rem;
          margin-bottom: 5px;
        }
        div {
          padding-top: 10px;
        }
        button {
          background-color: ${theme.defaultBg};
          color: ${theme.secondBgAndText};
          font-weight: bold;
          padding: 5px 15px;
          font-size: 1rem;
          border-radius: 10px;
          border: 0;
          transition: background-color 0.2s;
          cursor: pointer;
        }
        button:hover {
          background-color: ${theme.secondBgText};
        }
      `}</style>
    </div>
  )
}
