package models

import (
	"errors"
	"fmt"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/aws/aws-sdk-go/aws"
	DDB "github.com/aws/aws-sdk-go/service/dynamodb"
	. "github.com/nghichtu91/platform/share/planx/dynamodb"
)

const (
	HadRole = "had"
)

type AuthDynamoDB struct {
	*DynamoDB
}

type AuthUserShardInfo struct {
	GidSid        string `json:"gidsid" bson:"gid_sid,omitempty"`
	RoleName      string `json:"role_name" bson:"role_name,omitempty"`
	RoleLevel     string `json:"role_level" bson:"role_level,omitempty"`
	MainHero      string `json:"main_hero" bson:"main_hero,omitempty"`
	LastLoginTime string `json:"last_login_time" bson:"last_login_time,omitempty"`
	HistoryShard  string `json:"history_shard" bson:"history_shard,omitempty"`
	MainlineLevel string `json:"mainline_level" bson:"mainline_level,omitempty"`
	HeadIcon      string `json:"head_icon" bson:"head_icon,omitempty"`
	PlayerId      string `json:"player_id" bson:"player_id,omitempty"`
}

func (d *AuthDynamoDB) QueryUserShardInfo(table_name, uid string) ([]AuthUserShardInfo, error) {
	hash_typ, _, ok := d.GetHashType(table_name)
	if !ok {
		return nil, errors.New(fmt.Sprintf("No table Info, %s", table_name))
	}

	query_keys := make(map[string]*DDB.Condition, 4)
	query_keys[hash_typ] = &DDB.Condition{
		AttributeValueList: []*DDB.AttributeValue{CreateAttributeValue(uid)},
		ComparisonOperator: aws.String("EQ"),
	}

	in := &DDB.QueryInput{
		TableName:     aws.String(table_name),
		KeyConditions: query_keys,
	}

	query_out, err := d.Client().Query(in)
	if !ok {
		return nil, err
	}

	re := make([]AuthUserShardInfo, 0, len(query_out.Items))

	for _, i := range query_out.Items {
		uid, uok := i["Uid"]
		gid_sid, sok := i["gid_sid"]
		role_level, hok := i["role_level"]
		lastLoginTime := i["lastLoginTime"]
		mainlineLevel, mok := i["mainline_level"]
		head_icon, iok := i["head_icon"]

		if !uok || !sok || !hok || !mok || !iok {
			tilogs.L().Errorf("QueryUserShardInfo info error by %s!", uid)
			continue
		}

		rec := AuthUserShardInfo{}
		gs, sok := GetItemValue(gid_sid).(string)
		hr, hok := GetItemValue(role_level).(string)
		lt, lok := GetItemValue(lastLoginTime).(string)
		ml, mok := GetItemValue(mainlineLevel).(string)
		hi, iok := GetItemValue(head_icon).(string)
		if !sok || !hok || !lok || !mok || !iok {
			tilogs.L().Errorf("QueryUserShardInfo info typ err by %s for %v %v %v!", uid, gid_sid, role_level, lastLoginTime)
		}
		rec.GidSid = gs
		rec.RoleLevel = hr
		rec.LastLoginTime = lt
		rec.MainlineLevel = ml
		rec.HeadIcon = hi
		re = append(re, rec)
	}
	return re, nil
}

func (d *AuthDynamoDB) SetUserShardInfo(table_name, uid, gidSidStr string, hasRole, lastLoginTime string) error {
	hash_typ, range_typ, ok := d.GetHashType(table_name)
	if !ok {
		return errors.New(fmt.Sprintf("No table Info, %s", table_name))
	}

	updates := make(map[string]*DDB.AttributeValueUpdate, 2)
	v_a := CreateAttributeValue(hasRole)
	updates["has_role"] = &DDB.AttributeValueUpdate{
		Action: aws.String("PUT"),
		Value:  v_a,
	}

	v_l := CreateAttributeValue(lastLoginTime)
	updates["lastLoginTime"] = &DDB.AttributeValueUpdate{
		Action: aws.String("PUT"),
		Value:  v_l,
	}

	update_item := &DDB.UpdateItemInput{
		AttributeUpdates: updates,
		TableName:        aws.String(table_name),
		Key: map[string]*DDB.AttributeValue{
			hash_typ:  CreateAttributeValue(uid),
			range_typ: CreateAttributeValue(gidSidStr),
		},
		ReturnConsumedCapacity: aws.String("TOTAL"),
		Expected:               map[string]*DDB.ExpectedAttributeValue{},
		ReturnValues:           aws.String("UPDATED_NEW"),
	}
	_, err := d.Client().UpdateItem(update_item)
	return err
}
