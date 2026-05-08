import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

type SpeechRecognitionLike = {
  lang: string
  continuous: boolean
  interimResults: boolean
  start: () => void
  stop: () => void
  onresult: ((event: SpeechRecognitionEventLike) => void) | null
  onerror: ((event: SpeechRecognitionErrorEventLike) => void) | null
  onend: (() => void) | null
}

type SpeechRecognitionResultAlternativeLike = {
  transcript: string
}

type SpeechRecognitionResultLike = {
  0: SpeechRecognitionResultAlternativeLike
  isFinal: boolean
}

type SpeechRecognitionEventLike = {
  resultIndex: number
  results: ArrayLike<SpeechRecognitionResultLike>
}

type SpeechRecognitionErrorEventLike = {
  error: string
}

type SpeechRecognitionConstructor = new () => SpeechRecognitionLike

function getSpeechRecognitionConstructor(): SpeechRecognitionConstructor | null {
  const w = window as unknown as {
    SpeechRecognition?: SpeechRecognitionConstructor
    webkitSpeechRecognition?: SpeechRecognitionConstructor
  }

  return w.SpeechRecognition ?? w.webkitSpeechRecognition ?? null
}

export type SpeechRecognitionCallbacks = {
  onInterimText?: (text: string) => void
  onFinalText?: (text: string) => void
  onErrorText?: (message: string) => void
}

function formatSpeechRecognitionError(error: string) {
  switch (error) {
    case 'not-allowed':
    case 'service-not-allowed':
      return '未获得麦克风权限，请在浏览器设置中允许后重试'
    case 'audio-capture':
      return '未检测到可用麦克风设备'
    case 'no-speech':
      return '没有检测到语音输入，请靠近麦克风后重试'
    case 'network':
      return '语音识别服务网络不可用（部分网络环境下浏览器内置识别可能无法使用）'
    case 'aborted':
      return '语音识别已中止'
    default:
      return `语音识别失败：${error}`
  }
}

export function useSpeechRecognition(callbacks: SpeechRecognitionCallbacks) {
  const ctor = useMemo(
    () => (typeof window === 'undefined' ? null : getSpeechRecognitionConstructor()),
    [],
  )
  const isSupported = !!ctor

  const recognitionRef = useRef<SpeechRecognitionLike | null>(null)
  const shouldKeepListeningRef = useRef(false)

  const [isListening, setIsListening] = useState(false)

  const stop = useCallback(() => {
    shouldKeepListeningRef.current = false
    setIsListening(false)
    recognitionRef.current?.stop()
  }, [])

  const start = useCallback(() => {
    if (!ctor) {
      callbacks.onErrorText?.('当前浏览器不支持语音输入，请使用 Chrome/Edge')
      return
    }

    if (!recognitionRef.current) {
      const recognition = new ctor()
      recognition.lang = 'zh-CN'
      recognition.continuous = true
      recognition.interimResults = true

      recognition.onresult = (event: SpeechRecognitionEventLike) => {
        let interim = ''
        let final = ''

        for (let i = event.resultIndex; i < event.results.length; i++) {
          const result = event.results[i]
          const text = result?.[0]?.transcript ?? ''
          if (!text) continue
          if (result.isFinal) final += text
          else interim += text
        }

        if (interim) callbacks.onInterimText?.(interim)
        if (final) callbacks.onFinalText?.(final)
      }

      recognition.onerror = (event: SpeechRecognitionErrorEventLike) => {
        shouldKeepListeningRef.current = false
        setIsListening(false)
        callbacks.onErrorText?.(formatSpeechRecognitionError(event.error))
      }

      recognition.onend = () => {
        if (!shouldKeepListeningRef.current) {
          setIsListening(false)
          return
        }

        try {
          recognition.start()
        } catch {
          void 0
        }
      }

      recognitionRef.current = recognition
    }

    shouldKeepListeningRef.current = true
    setIsListening(true)

    try {
      recognitionRef.current.start()
    } catch {
      void 0
    }
  }, [callbacks, ctor])

  useEffect(() => stop, [stop])

  return { isSupported, isListening, start, stop }
}
