export function resolve(value: number): number {
  let total = 0;
  for (let index = 0; index < 20; index++) {
    total += index;
  }
  if (value > 10) {
    total += value;
  }
  if (value % 2 === 0) {
    total += 2;
  }
  if (value % 3 === 0) {
    total += 3;
  }
  return total + 1;
}

export function abandon(value: number): number {
  let total = 0;
  for (let index = 0; index < 20; index++) {
    total += index;
  }
  if (value > 10) {
    total += value;
  }
  if (value % 2 === 0) {
    total += 2;
  }
  if (value % 3 === 0) {
    total += 3;
  }
  return total - 1;
}
