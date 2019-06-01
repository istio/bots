// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

type HandlerFn<T> = (data?: T) => void;

interface ILiteEvent<T> {
    on(handler: HandlerFn<T>): void;
    off(handler: HandlerFn<T>): void;
}

class LiteEvent<T> implements ILiteEvent<T> {
    private handlers: Array<HandlerFn<T>> = new Array<HandlerFn<T>>();

    public on(handler: HandlerFn<T>): void {
        this.handlers.push(handler);
    }

    public off(handler: HandlerFn<T>): void {
        this.handlers = this.handlers.filter(h => h !== handler);
    }

    public trigger(data?: T) {
        this.handlers.slice(0).forEach(h => h(data));
    }

    public expose(): ILiteEvent<T> {
        return this;
    }
}
