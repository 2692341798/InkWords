export function shouldResetPolishState(errorMessage: string): boolean {
  return errorMessage !== 'done' && errorMessage !== 'aborted'
}

