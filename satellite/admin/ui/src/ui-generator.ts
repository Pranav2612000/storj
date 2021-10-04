export interface Operation {
  // name is the operation name.
  name: string;
  // desc is the description of the operation.
  desc: string;
  // params is an array of tuples where each tuple corresponds to one parameter
  // of 'func'. Each tuple has 2 elements, the first is the parameters name and
  // the second how it's mapped to the UI. The orders must match the order of
  // the 'func' parameters.
  // The parameter's name is what is going to be show next to the input field,
  // so it has to be descriptive for the users to know that they have to set.
  params: [string, ParamUI][];
  // func is the API function call. They always have to return a promise which
  // resolves with an object or null.
  // On a resolved promise, an object is the response body of an API call and
  // null is used when the API operation doesn't return any payload (e.g. PUT
  // operations).
  func: (...p: any) => Promise<object | null>;
}

type ParamUI = InputText | Select | Textarea;

export class InputText {
  constructor(
    public readonly type: "checkbox" | "email" | "number" | "password" | "text",
    public readonly required: boolean
  ) {}
}

export class Select {
  constructor(
    public readonly multiple: boolean,
    public readonly required: boolean,
    public readonly options: {
      text: string;
      value: boolean | number | string;
    }[]
  ) {}
}

export class Textarea {
  constructor(public readonly required: boolean) {}
}
