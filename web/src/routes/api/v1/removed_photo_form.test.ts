import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";
import { PUT as createTrail } from "./trail/form/+server";
import { POST as updateTrail } from "./trail/form/[id]/+server";
import { PUT as createLog } from "./summit-log/form/+server";
import { POST as updateLog } from "./summit-log/form/[id]/+server";

describe.each([
    ["trail", createTrail, "PUT"], ["trail", updateTrail, "POST"],
    ["summit_log", createLog, "PUT"], ["summit_log", updateLog, "POST"],
] as const)("removed photo parts on %s %s", (kind, handler, method) => {
    it.each(["photos", "photos+", "+photos", "photos-", "-photos", "@jsonPayload"])("rejects %s before any record write", async field => {
        const data = new FormData();
        data.set("name", "Changed name");
        data.set(field, field === "@jsonPayload" ? JSON.stringify({ "photos+": ["image.jpg"] }) : new File(["photo bytes"], "image.jpg"));
        const collection = vi.fn();
        const url = new URL("https://a.example/api/v1/form");
        const event = { request: new Request(url, { method, body: data }), url, params: { id: "record000000001" }, locals: { user: { id: "user" }, pb: { collection } } } as unknown as RequestEvent;
        const response = await handler(event);
        expect(response.status).toBe(400);
        expect(await response.json()).toEqual({ message: `${kind}_photos_form_field_removed` });
        expect(collection).not.toHaveBeenCalled();
    });
});


describe.each([
 ["trail", createTrail, "PUT"], ["trail", updateTrail, "POST"],
 ["summit_log", createLog, "PUT"], ["summit_log", updateLog, "POST"],
] as const)("embedded photo parts on %s %s", (kind, handler, method) => {
 it.each(["photos", "photos+", "+photos", "photos-", "-photos"])("rejects %s alongside GPX before any write", async field => {
  const data = new FormData();
  data.set("gpx", new File(["route"], "route.gpx"));
  data.append("@jsonPayload", JSON.stringify({name:"Allowed metadata"}));
  data.append("@jsonPayload", JSON.stringify({[field]: []}));
  const collection = vi.fn();
  const url = new URL("https://a.example/api/v1/form");
  const event = {request:new Request(url,{method,body:data}), url, params:{id:"record000000001"},locals:{user:{id:"user"},pb:{collection}}} as unknown as RequestEvent;
  const response = await handler(event);
  expect(response.status).toBe(400);
  expect(await response.json()).toEqual({message:`${kind}_photos_form_field_removed`});
  expect(collection).not.toHaveBeenCalled();
 });
});
