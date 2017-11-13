package message

import (
    "encoding/json"
)


/* A Simple function to verify error */
func CheckError(err error) {
    if err  != nil {
        panic(err)
    }
}


type Message struct {
    ID_           int        `json:"id"`           //id сообщения
    Type_         string     `json:"type"`         //тип сообщения
    Sender_       int        `json:"sender"`       //отправитель
    Origin_       int        `json:"origin"`       //исходный узел-отправитель
    Data_         string     `json:"data"`         //строка, содрежащая данные
}


func (msg Message) ToJsonMsg() []byte {
    buf, err := json.Marshal(msg)
    CheckError(err)
    return buf
}

func FromJsonMsg(buffer []byte) Message {
    var msg Message
    err := json.Unmarshal(buffer, &msg)
    CheckError(err)
    return msg
}
