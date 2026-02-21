import ky from "ky";

export const apiClient = ky.create({
  prefixUrl: "",
  timeout: 30_000,
  hooks: {
    beforeError: [
      async (error) => {
        const { response } = error;
        if (response) {
          try {
            const body = await response.json() as Record<string, unknown>;
            error.message = (body.error as string) || error.message;
            if (body.details) {
              (error as unknown as Record<string, unknown>).details = body.details;
            }
          } catch {
            // response wasn't JSON
          }
        }
        return error;
      },
    ],
  },
});
