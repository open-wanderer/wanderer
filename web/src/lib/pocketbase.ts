import { browser } from '$app/env';
import { PUBLIC_POCKETBASE_URL } from '$app/env/public'
import PocketBase from 'pocketbase'

let pb: PocketBase;

export function getPb(): PocketBase {
  if(!browser) {
    throw Error("Only available client side!")
  }
  if (!pb) {
    pb = new PocketBase(PUBLIC_POCKETBASE_URL);
  }
  return pb;
}