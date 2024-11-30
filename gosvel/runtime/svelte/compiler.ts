import { compile as compileSvelte } from "svelte/compiler"

type Input = {
  code: string
  path: string,
  rootDir: string,
  target: false | "client" | "server" | undefined
  dev: boolean
  css: "injected" | "external" | undefined
}

// Capitalized for Go
type Output =
  | {
      JS: string
      CSS: string
    }
  | {
      Error: {
        Path: string
        Name: string
        Message: string
        Stack?: string
      }
    }

// Compile svelte code
export function compile(input: Input): string {
  const { code, path, target, dev, css, rootDir } = input
  const svelte = compileSvelte(code, {
    filename: path,
    generate: target,
    dev: dev,
    css: css,
    rootDir: rootDir,
    runes: true,
    preserveWhitespace: false,
  })
  return JSON.stringify({
    CSS: svelte.css?.code,
    JS: svelte.js.code,
  } as Output)
}