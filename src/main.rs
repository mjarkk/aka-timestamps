use std::env;
use std::fmt::Display;
use std::fs;
use std::io;
use std::io::{prelude::*, BufReader};
use std::process;

struct Time {}

struct TextAtTime {
    text: String,
    at: Time,
}

struct QuestionType {
    full: String,
    searchable: String,
    shortent: String,
}

struct DetectedTimeStamp {
    question: QuestionType,
    at_str: String,
    found: bool,
}

fn check<T, Y>(input: Result<T, Y>) -> T
where
    Y: Display,
{
    match input {
        Err(err) => {
            println!("Err: {}", err);
            process::exit(1);
        }
        Ok(v) => v,
    }
}

fn read_file(filename: &str) -> Result<String, String> {
    let mut contents = String::new();
    let file = match fs::File::open(filename) {
        Err(err) => return Err(err.to_string()),
        Ok(file) => file,
    };
    let mut buf_reader = BufReader::new(file);
    if let Err(err) = buf_reader.read_to_string(&mut contents) {
        return Err(err.to_string());
    }
    Ok(contents)
}

fn cleanup_and_prepair() -> io::Result<()> {
    match fs::remove_dir_all(".vid-meta") {
        Ok(_) => {}
        Err(_) => {}
    };
    fs::create_dir(".vid-meta")
}

fn download_video_meta() -> io::Result<()> {
    let cwd = format!("{}/.vid-meta", env::current_dir()?.to_str().unwrap());
    process::Command::new("../youtube-dl")
        .args(&[
            &env::args().last().unwrap().to_string(),
            "--write-auto-sub",
            "--write-description",
            "--output",
            "vid",
            "--skip-download",
        ])
        .current_dir(cwd)
        .output()?;

    Ok(())
}

fn extract_comments() -> Result<Vec<String>, String> {
    let mut description = read_file(".vid-meta/vid.description")?;

    for i in 1..11 {
        let mut to_replace = String::new();
        for _ in 0..i {
            to_replace.push_str(" ");
        }
        description = description.replace(&format!("\n{}\n", to_replace), "\n\n");
    }
    for _ in 0..4 {
        description = description.replace("\n\n\n", "\n\n");
    }

    let mut matched = vec![];
    let mut min_number: isize = 1;

    let parts: Vec<&str> = description.split("\n\n").collect();
    for mut part in parts {
        part = part.trim();
        if part.len() < 10 {
            continue;
        }

        let mut part_vec: Vec<char> = part.chars().collect();
        if !"123456789".contains(part_vec[0]) {
            let sub_parts: Vec<&str> = part.splitn(2, "\n").collect();
            if sub_parts.len() != 2 {
                continue;
            }
            part = sub_parts[1];
            part_vec = part.chars().collect();
            if !"123456789".contains(part_vec[0]) {
                continue;
            }
        }

        let mut question_number_str = format!("{}", part_vec[0]);
        if "1234567890".contains(part_vec[1]) {
            question_number_str = format!("{}{}", question_number_str, part_vec[1]);
        }
        let question_number = match question_number_str.parse::<isize>() {
            Err(_) => continue,
            Ok(v) => v,
        };
        if question_number < min_number - 4 || question_number > min_number + 4 {
            continue;
        }

        matched.push(part.to_string());
        min_number = question_number;
    }

    Ok(matched)
}

fn extract_subtitles() -> Result<(String, String), String> {
    let mut subtitles = read_file(".vid-meta/vid.en.vtt")?;

    Ok((String::new(), String::new()))
}

fn main() {
    println!("Cleaning up and preparing..");
    check(cleanup_and_prepair());

    println!("Downloading video meta data..");
    check(download_video_meta());

    println!("Extracting subtitles..");
    let (words, wordsMap) = check(extract_subtitles());

    println!("Extracting comments..");
    let linesThatMightBeQuestions = check(extract_comments());
}
