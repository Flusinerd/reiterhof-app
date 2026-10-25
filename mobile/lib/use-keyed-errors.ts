import { useCallback, useState } from "react";

/**
 * The errors of writes on the rows of a list, by row id. Each one is shown under its own row: an error above
 * a long list is out of sight when the user taps further down (docs/design-system.md, "Feedback").
 *
 * @example
 * const [errors, setError] = useKeyedErrors();
 * setError(item.id, null);                         // when the write starts
 * setError(item.id, errorMessage(e));              // when it fails
 * <Row error={errors[item.id]} />
 */
export function useKeyedErrors() {
  const [errors, setErrors] = useState<Record<string, string>>({});
  const setError = useCallback(
    (id: string, text: string | null) =>
      setErrors(({ [id]: _old, ...rest }) => (text === null ? rest : { ...rest, [id]: text })),
    [],
  );
  return [errors, setError] as const;
}
