import Link from './link';

type GpxEmail = { $: { id: string, domain: string } };

export default class Person {
  name: string;
  email?: GpxEmail;
  link?: Link;
  constructor(object: { name: string, email?: string | GpxEmail, link?: Link }) {
    this.name = object.name;
    if (object.email) {
      if (typeof object.email === 'string') {
        const [id, domain] = object.email.split('@');
        // GPX 1.1 requires <email id="user" domain="example.com"/>, not
        // a plain-text email element, or GpxReader-based clients (e.g. the
        // mobile app) fail to parse the file.
        if (id && domain) {
          this.email = { $: { id, domain } };
        }
      } else {
        this.email = object.email;
      }
    }
    if (object.link) {
      this.link = new Link(object.link);
    }
  }
}