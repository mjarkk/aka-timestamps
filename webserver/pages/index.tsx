import Head from 'next/head'
import { useEffect, useState } from 'react'

const url = (path: string) => (window.location.host == 'localhost:3000' ? 'http://localhost:9090' : 'https://api.aka-podcast.mkopenga.com.mkopenga.com') + path

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

export default function Home() {
  const [eps, setEps] = useState([] as EP[])

  useEffect(() => {
    fetch(url('/eps'))
      .then(r => r.json())
      .then(eps => setEps(eps))
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
          {eps.map((ep, key) => <EpBlock key={key} ep={ep} />)}
          <p className="info">The code of this tool is open source and can be found <a href="https://github.com/mjarkk/aka-timestamps">here</a></p>
        </div>
      </main>
      <style jsx global>{`
        * {
          padding: 0px;
          margin: 0px;
        }
        body {
          font-size: 18px;
          font-family: sans-serif;
          background-color: #fafcc6;
          color: #e3b4af;
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
          color: #e3b4af;
          transition: color 0.1s;
        }
        .info:hover a {
          color: #bb918c;
        }
      `}</style>
    </>
  )
}

function EpBlock({ ep }: { ep: EP }) {
  return (
    <div className="ep-block">
      <h3>{ep.name}</h3>
      {ep.foundResults?.questions && ep.foundResults?.timeStamp
        ? <Timestamps res={ep.foundResults} />
        : <div className="meta">
          <p>{
            !ep.foundDescription ? `Unable to read description of episode`
              : !ep.foundVTT ? `Unable to read description of episode`
                : ep.foundResults?.err ? `Oh wired error: ${ep.foundResults?.err}`
                  : `Unable to get timestamps for this episode`
          }</p>
        </div>
      }
      <style jsx>{`
        .ep-block {
          background-color: #e3b4af;
          padding: 20px;
          margin: 40px 0px;
          border-radius: 10px;
          color: #5d07fe;
          overflow: hidden;
        }
        .ep-block h3 {
          margin-bottom: 10px;
        }
      `}</style>
    </div>
  )
}

function Timestamps({ res }: { res: AnylizeResults }) {
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

  return (
    <div className="time-stamps">
      {timestamps.map((timestamp, key) =>
        <p key={key}>{timestamp.atStr} {question(timestamp.questionIdx).shortent}</p>
      )}
      <style jsx>{`
        p {
          font-size: 0.9rem;
          margin-bottom: 5px;
        }
      `}</style>
    </div>
  )
}
