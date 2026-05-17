export type UserName = {
  readonly firstName: string;
  readonly lastName: string;
};

export function formatUser(input: UserName): string {
  return `${input.firstName} ${input.lastName}`;
}
