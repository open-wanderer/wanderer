export default class Link {
  $: {
    href?: string;
  }
  text: string;
  type: string;
  constructor(object: any) {
    this.$ = {};
    this.$.href = object?.$?.href || object?.href;
    this.text = object?.text;
    this.type = object?.type;
  }
}