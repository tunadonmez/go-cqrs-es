function getEnv(value: string | undefined, name: string) {
  if (!value) {
    throw new Error(`${name} is not configured.`);
  }
  return value.replace(/\/+$/, "");
}

export const appConfig = {
  commandApiUrl: getEnv(process.env.NEXT_PUBLIC_COMMAND_API_URL, "NEXT_PUBLIC_COMMAND_API_URL"),
  queryApiUrl: getEnv(process.env.NEXT_PUBLIC_QUERY_API_URL, "NEXT_PUBLIC_QUERY_API_URL")
};
