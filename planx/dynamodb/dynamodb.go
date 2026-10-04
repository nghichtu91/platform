package dynamodb

import (
	"errors"
	"os"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"fmt"

	"time"

	"github.com/aws/aws-sdk-go/aws"
	DDB "github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/cenk/backoff"
	"github.com/nghichtu91/platform/share/planx/awshelper"
)

type DynamoDB struct {
	client        *DDB.DynamoDB
	noRetryClient *DDB.DynamoDB
	//TODO: YZH DynamoDB table_des的存在很奇怪
	table_des map[string]TableInfo
}

func (d *DynamoDB) Client() *DDB.DynamoDB {
	return d.client
}

func (d *DynamoDB) Connect(
	region,
	accessKeyID,
	secretAccessKey,
	sessionToken string) error {

	if region == "" {
		region = os.Getenv("AWS_REGION")
	}
	// For https://github.com/aws/aws-sdk-ruby/issues/243
	mySession := awshelper.CreateAWSSession(region, accessKeyID, secretAccessKey, 3)
	d.client = DDB.New(mySession)

	noRetrySession := awshelper.CreateAWSSession(region, accessKeyID, secretAccessKey, 0)
	d.noRetryClient = DDB.New(noRetrySession)

	tilogs.L().Infof("DynamoDB Connect ok")
	return nil
}

func (d *DynamoDB) Set(table_name string,
	hash, range_value, value interface{}) error {
	hash_typ, range_typ, ok := d.GetHashType(table_name)
	if !ok {
		return errors.New(fmt.Sprintf("No table Info, %s", table_name))
	}

	put_input := &DDB.PutItemInput{
		TableName: aws.String(table_name),
		Item: map[string]*DDB.AttributeValue{
			hash_typ:  CreateAttributeValue(hash),
			range_typ: CreateAttributeValue(range_value),
			ValueType: CreateAttributeValue(value),
		},
		ReturnConsumedCapacity: aws.String("TOTAL"),
		Expected:               map[string]*DDB.ExpectedAttributeValue{},
	}
	_, err := d.client.PutItem(put_input)
	return err
}

func (d *DynamoDB) Get(table_name string,
	hash, range_value interface{}) (map[string]interface{}, error) {
	hash_typ, range_typ, ok := d.GetHashType(table_name)
	if !ok {
		return nil, errors.New(fmt.Sprintf("No table Info, %s", table_name))
	}

	get_item := &DDB.GetItemInput{
		ConsistentRead: aws.Bool(true),
		TableName:      aws.String(table_name),
		Key: map[string]*DDB.AttributeValue{
			hash_typ:  CreateAttributeValue(hash),
			range_typ: CreateAttributeValue(range_value),
		},
	}
	get_item_out, err := d.client.GetItem(get_item)
	if err != nil {
		return nil, err
	} else {
		re := make(map[string]interface{}, len(get_item_out.Item))
		for i, v := range get_item_out.Item {
			re[i] = GetItemValue(v)
		}
		return re, nil
	}
}

func (d *DynamoDB) SetByHash(table_name string, hash, value interface{}) error {
	hash_typ, _, ok := d.GetHashType(table_name)
	if !ok {
		return errors.New(fmt.Sprintf("No table Info, %s", table_name))
	}

	put_input := &DDB.PutItemInput{
		TableName: aws.String(table_name),
		Item: map[string]*DDB.AttributeValue{
			hash_typ:  CreateAttributeValue(hash),
			ValueType: CreateAttributeValue(value),
		},
		ReturnConsumedCapacity: aws.String("TOTAL"),
		Expected:               map[string]*DDB.ExpectedAttributeValue{},
	}
	_, err := d.client.PutItem(put_input)
	return err
}

func (d *DynamoDB) SetByHashB(table_name string, hash interface{}, value []byte) error {
	hash_typ, _, ok := d.GetHashType(table_name)
	if !ok {
		return errors.New(fmt.Sprintf("No table Info, %s", table_name))
	}

	tilogs.L().Warnf("SetByHashB %v", value)

	put_input := &DDB.PutItemInput{
		TableName: aws.String(table_name),
		Item: map[string]*DDB.AttributeValue{
			hash_typ: CreateAttributeValue(hash),
			ValueType: {
				B: value,
			},
		},
		ReturnConsumedCapacity: aws.String("TOTAL"),
		Expected:               map[string]*DDB.ExpectedAttributeValue{},
	}
	_, err := d.client.PutItem(put_input)
	return err
}

func (d *DynamoDB) DelKey(table_name string, hash interface{}) error {
	hash_typ, _, ok := d.GetHashType(table_name)
	if !ok {
		return errors.New(fmt.Sprintf("No table Info, %s", table_name))
	}

	tilogs.L().Warnf("Del Key %v from %s", hash, table_name)

	put_input := &DDB.DeleteItemInput{
		TableName: aws.String(table_name),
		Key: map[string]*DDB.AttributeValue{
			hash_typ: CreateAttributeValue(hash),
		},
		ReturnConsumedCapacity: aws.String("TOTAL"),
		Expected:               map[string]*DDB.ExpectedAttributeValue{},
	}
	_, err := d.client.DeleteItem(put_input)
	return err
}

func (d *DynamoDB) DelKeyRange(table_name string, hash interface{}, rangeKey interface{}) error {
	hash_typ, ramge_typ, ok := d.GetHashType(table_name)
	if !ok {
		return errors.New(fmt.Sprintf("No table Info, %s", table_name))
	}

	tilogs.L().Warnf("Del Key %v from %s", hash, table_name)

	put_input := &DDB.DeleteItemInput{
		TableName: aws.String(table_name),
		Key: map[string]*DDB.AttributeValue{
			hash_typ:  CreateAttributeValue(hash),
			ramge_typ: CreateAttributeValue(rangeKey),
		},
		ReturnConsumedCapacity: aws.String("TOTAL"),
		Expected:               map[string]*DDB.ExpectedAttributeValue{},
	}
	_, err := d.client.DeleteItem(put_input)
	return err
}

/* 上层业务要遵守dynamodb batch write某些限制条件
  	input := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			"Music": {
				{
					PutRequest: &dynamodb.PutRequest{
						Item: map[string]*dynamodb.AttributeValue{
							"AlbumTitle": {
								S: aws.String("Somewhat Famous"),
							},
							"Artist": {
								S: aws.String("No One You Know"),
							},
							"SongTitle": {
								S: aws.String("Call Me Today"),
							},
						},
					},
				},
				{
					PutRequest: &dynamodb.PutRequest{
						Item: map[string]*dynamodb.AttributeValue{
							"AlbumTitle": {
								S: aws.String("Songs About Life"),
							},
							"Artist": {
								S: aws.String("Acme Band"),
							},
							"SongTitle": {
								S: aws.String("Happy Day"),
							},
						},
					},
				},
			},
		}
对应的输入是 BatchSetByHashM("Music", {
			{
				"AlbumTitle": {
					S: aws.String("Somewhat Famous"),
				},
				"Artist": {
					S: aws.String("No One You Know"),
				},
				"SongTitle": {
					S: aws.String("Call Me Today"),
				},
			},
			{
				"AlbumTitle": {
					S: aws.String("Songs About Life"),
				},
				"Artist": {
					S: aws.String("Acme Band"),
				},
				"SongTitle": {
					S: aws.String("Happy Day"),
				},
			})
*/
func (d *DynamoDB) BatchSetByHashM(table_name string, values []map[string]interface{}) (error, int) {
	PutRequests := make([]*DDB.WriteRequest, 0, len(values))
	for _, value := range values {
		item := make(map[string]*DDB.AttributeValue, len(value))
		for typ, v := range value {
			item[typ] = CreateAttributeValue(v)
		}
		PutRequests = append(PutRequests, &DDB.WriteRequest{
			PutRequest: &DDB.PutRequest{
				Item: item,
			},
		})
	}
	params := &DDB.BatchWriteItemInput{
		RequestItems: map[string][]*DDB.WriteRequest{ // Required
			table_name: PutRequests,
		},
		ReturnConsumedCapacity: aws.String("TOTAL"),
	}

	resp, err := d.client.BatchWriteItem(params)
	if err != nil {
		tilogs.L().Errorf("BatchWriteItem Err by %s", err.Error())
		return err, len(values)
	}
	if len(resp.UnprocessedItems) > 0 {
		b := NewExponentialBackOffSleepFirst()
		err = backoff.RetryNotify(func() error {
			newPutRequests := PutRequests[0:0]
			for _, v := range resp.UnprocessedItems {
				for _, failed := range v {
					if failed.PutRequest != nil || failed.DeleteRequest != nil {
						newPutRequests = append(newPutRequests, failed)
					}
				}
			}
			nparams := &DDB.BatchWriteItemInput{
				RequestItems: map[string][]*DDB.WriteRequest{ // Required
					table_name: newPutRequests,
				},
				ReturnConsumedCapacity: aws.String("TOTAL"),
			}
			nresp, nerr := d.client.BatchWriteItem(nparams)
			resp = nresp
			if nerr != nil {
				return nerr
			}

			if len(resp.UnprocessedItems) > 0 {
				return fmt.Errorf("UnprocessedItems")
			}
			return nil
		}, b, func(e error, d time.Duration) {
		})
		return err, len(resp.UnprocessedItems)
	}
	return nil, 0
}

func (d *DynamoDB) SetByHashM(table_name string, hash interface{},
	values map[string]interface{}) error {
	hash_typ, _, ok := d.GetHashType(table_name)
	if !ok {
		return errors.New(fmt.Sprintf("No table Info, %s", table_name))
	}

	items := make(map[string]*DDB.AttributeValue, len(values)+1)
	items[hash_typ] = CreateAttributeValue(hash)

	for typ, v := range values {
		items[typ] = CreateAttributeValue(v)
	}

	put_input := &DDB.PutItemInput{
		TableName:              aws.String(table_name),
		Item:                   items,
		ReturnConsumedCapacity: aws.String("TOTAL"),
		Expected:               map[string]*DDB.ExpectedAttributeValue{},
	}
	_, err := d.client.PutItem(put_input)
	return err
}

func (d *DynamoDB) SetByHashM_IfNoExist(table_name string, hash interface{},
	values map[string]interface{}) error {
	hash_typ, _, ok := d.GetHashType(table_name)
	if !ok {
		return errors.New(fmt.Sprintf("No table Info, %s", table_name))
	}

	items := make(map[string]*DDB.AttributeValue, len(values)+1)
	items[hash_typ] = CreateAttributeValue(hash)

	for typ, v := range values {
		items[typ] = CreateAttributeValue(v)
	}

	put_input := &DDB.PutItemInput{
		TableName:              aws.String(table_name),
		Item:                   items,
		ReturnConsumedCapacity: aws.String("TOTAL"),
		//Expected:               map[string]*DDB.ExpectedAttributeValue{},

		ExpressionAttributeNames: map[string]*string{
			"#hash_typ": aws.String(fmt.Sprintf("%s", hash_typ)),
		},
		ConditionExpression: aws.String("attribute_not_exists(#hash_typ)"),
	}
	_, err := d.client.PutItem(put_input)
	return err
}

func (d *DynamoDB) GetByHashM(table_name string, hash interface{}) (map[string]interface{}, error) {
	hash_typ, _, ok := d.GetHashType(table_name)
	if !ok {
		return nil, errors.New(fmt.Sprintf("No table Info, %s", table_name))
	}

	get_item := &DDB.GetItemInput{
		ConsistentRead: aws.Bool(true),
		TableName:      aws.String(table_name),
		Key: map[string]*DDB.AttributeValue{
			hash_typ: CreateAttributeValue(hash),
		},
	}
	get_item_out, err := d.client.GetItem(get_item)
	if err != nil {
		return nil, err
	} else {
		re := make(map[string]interface{}, len(get_item_out.Item))
		for i, v := range get_item_out.Item {
			re[i] = GetItemValue(v)
		}
		return re, nil
	}
}

func (d *DynamoDB) GetByHash(table_name string, hash interface{}) (interface{}, error) {
	hash_typ, _, ok := d.GetHashType(table_name)
	if !ok {
		return "", errors.New(fmt.Sprintf("No table Info, %s", table_name))
	}

	get_item := &DDB.GetItemInput{
		ConsistentRead: aws.Bool(true),
		TableName:      aws.String(table_name),
		Key: map[string]*DDB.AttributeValue{
			hash_typ: CreateAttributeValue(hash),
		},
	}
	get_item_out, err := d.client.GetItem(get_item)
	if get_item_out == nil || get_item_out.Item == nil {
		return nil, err
	}
	items := get_item_out.Item

	//tilogs.L().Warnf("GetByHash %v %v %v", items[ValueType].B, table_name, hash_typ)

	return GetItemValue(items[ValueType]), err
}

func (d *DynamoDB) QueryByHash(table_name string,
	hash interface{}) ([]interface{}, error) {

	hash_typ, _, ok := d.GetHashType(table_name)
	if !ok {
		return []interface{}{}, errors.New(fmt.Sprintf("No table Info, %s", table_name))
	}

	query_keys := make(map[string]*DDB.Condition)
	query_keys[hash_typ] = &DDB.Condition{
		AttributeValueList: []*DDB.AttributeValue{CreateAttributeValue(hash)},
		ComparisonOperator: aws.String("EQ"),
	}

	in := &DDB.QueryInput{
		TableName:     aws.String(table_name),
		KeyConditions: query_keys,
	}

	query_out, err := d.client.Query(in)

	re := make([]interface{}, 0, len(query_out.Items))

	for _, value := range query_out.Items {
		re = append(re, GetItemValue(value[ValueType]))

	}
	return re[:], err
}

func (d *DynamoDB) QueryByAccountId(table_name, indexname, accountId string) (map[string]interface{}, error) {
	params := &DDB.QueryInput{

		TableName:      aws.String(table_name),
		IndexName:      aws.String(indexname),
		ConsistentRead: aws.Bool(false),
		ExpressionAttributeValues: map[string]*DDB.AttributeValue{
			":u": CreateAttributeValue(accountId),
		},
		KeyConditionExpression: aws.String("user_id = :u"),
		Limit:                  aws.Int64(1000),
		ScanIndexForward:       aws.Bool(true),
	}

	resp, err := d.Client().Query(params)
	if len(resp.Items) <= 0 {
		return nil, err
	}
	re := make(map[string]interface{}, 0)

	recall_id, ok := resp.Items[0]["recall_id"]
	if ok {
		re["recallid"] = GetItemValue(recall_id).(string)
	}
	recallNum, ok := resp.Items[0]["recall_num"]
	if ok {
		re["recall_num"] = GetItemValue(recallNum).(int64)
	}
	userId, ok := resp.Items[0]["user_id"]
	if ok {
		re["user_id"] = GetItemValue(userId).(string)
	}
	tilogs.L().Debugf("QueryByAccountId %v", re)
	return re, err

}

func (d *DynamoDB) UpdateByHash(
	table_name string,
	hash interface{},
	values map[string]interface{}) error {

	hash_typ, _, ok := d.GetHashType(table_name)
	if !ok {
		return errors.New(fmt.Sprintf("No table Info, %s", table_name))
	}

	updates := make(map[string]*DDB.AttributeValueUpdate, len(values))
	for typ, v := range values {
		v_a := CreateAttributeValue(v)
		updates[typ] = &DDB.AttributeValueUpdate{
			Action: aws.String("PUT"),
			Value:  v_a,
		}
	}

	update_item := &DDB.UpdateItemInput{
		AttributeUpdates: updates,
		TableName:        aws.String(table_name),
		Key: map[string]*DDB.AttributeValue{
			hash_typ: CreateAttributeValue(hash),
		},
		ReturnConsumedCapacity: aws.String("TOTAL"),
		Expected:               map[string]*DDB.ExpectedAttributeValue{},
		ReturnValues:           aws.String("UPDATED_NEW"),
	}
	_, err := d.client.UpdateItem(update_item)
	return err
}

func (d *DynamoDB) Incr(table_name string,
	hash interface{},
	item_type string,
	value int64) (int64, error) {
	hash_typ, _, ok := d.GetHashType(table_name)
	if !ok {
		return -1, errors.New(fmt.Sprintf("No table Info, %s", table_name))
	}

	update_input := &DDB.UpdateItemInput{
		TableName: aws.String(table_name),
		Key: map[string]*DDB.AttributeValue{
			hash_typ: CreateAttributeValue(hash),
		},
		ExpressionAttributeNames: map[string]*string{
			"#Q": &item_type,
		},
		ExpressionAttributeValues: map[string]*DDB.AttributeValue{
			":incr": CreateAttributeValue(value),
		},
		UpdateExpression: aws.String("SET #Q = #Q + :incr"),

		ReturnConsumedCapacity: aws.String("TOTAL"),
		ReturnValues:           aws.String("UPDATED_NEW"),
	}
	update_output, err := d.client.UpdateItem(update_input)
	if err != nil {
		return -1, err
	}
	v, ok := update_output.Attributes[item_type]
	if !ok {
		return -2, errors.New("No type Return")
	}
	v_any := GetItemValue(v)
	v_int, ok := v_any.(int64)
	if !ok {
		return -3, errors.New("type not int64")
	}
	return v_int, nil
}

type ScanHander func(idx int, key interface{}, data interface{}) error

func (d *DynamoDB) rescan(ticker *backoff.Ticker, params *DDB.ScanInput) (*DDB.ScanOutput, error) {
	var err error
	for range ticker.C {
		output, err := d.client.Scan(params)
		if err == nil {
			ticker.Stop()
			return output, err
		}
	}

	ticker.Stop()
	return nil, err
}

func (d *DynamoDB) Scan(name string, scan_len int64, hander ScanHander, b *backoff.ExponentialBackOff) error {
	hash_typ, _, ok := d.GetHashType(name)
	if !ok {
		return errors.New(fmt.Sprintf("No table Info, %s", name))
	}

	var last_key map[string]*DDB.AttributeValue
	var all int = 0
	for {
		params := &DDB.ScanInput{
			TableName: aws.String(name),
			AttributesToGet: []*string{
				aws.String(hash_typ),
				aws.String(ValueType),
			},

			Limit: aws.Int64(scan_len),
		}
		if last_key != nil {
			params.ExclusiveStartKey = last_key
		}
		output, err := d.client.Scan(params)
		if err != nil {
			ticker := backoff.NewTicker(b)
			output, err = d.rescan(ticker, params)
		}

		if err != nil {
			return err
		}

		for _, item := range output.Items {
			all++
			key := GetItemValue(item[hash_typ])
			value := GetItemValue(item[ValueType])

			herr := hander(all, key, value)
			if herr != nil {
				return herr
			}
		}

		last_key = output.LastEvaluatedKey
		if last_key == nil {
			break
		}

	}

	return nil
}

func (d *DynamoDB) ScanBy(tablename, searchname, colname string) (string, error) {
	_, _, ok := d.GetHashType(tablename)
	if !ok {
		return "", errors.New(fmt.Sprintf("No table Info, %s", tablename))
	}
	var result string
	var lastKey map[string]*DDB.AttributeValue
	for {
		params := &DDB.ScanInput{
			TableName:         aws.String(tablename), // Required
			ExclusiveStartKey: lastKey,
			ConsistentRead:    aws.Bool(true),
			ExpressionAttributeNames: map[string]*string{
				"#userId": aws.String(colname), // Required
			},
			ExpressionAttributeValues: map[string]*DDB.AttributeValue{
				":userId": {S: aws.String(searchname)},
			},
			FilterExpression: aws.String("#userId = :userId"),
			Limit:            aws.Int64(1000),
			Select:           aws.String(DDB.SelectAllAttributes),
		}

		//fmt.Println(params.String())

		resp, err := d.client.Scan(params)

		if err != nil {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
			return "", err
		}
		result = resp.String()
		// Pretty-print the response data.

		lastKey = resp.LastEvaluatedKey
		if lastKey == nil {
			break
		}
	}
	return result, nil
}

type DynamoKV struct {
	K interface{}
	V interface{}
}

// need to go ParallelScan()
func (d *DynamoDB) ParallelScan(
	name string,
	scan_len, scan_idx, scan_all int64,
	channel chan DynamoKV,
	b *backoff.ExponentialBackOff) error {

	hash_typ, _, ok := d.GetHashType(name)
	if !ok {
		return errors.New(fmt.Sprintf("No table Info, %s", name))
	}

	var last_key map[string]*DDB.AttributeValue
	var all int = 0
	for {

		params := &DDB.ScanInput{
			TableName: aws.String(name),
			AttributesToGet: []*string{
				aws.String(hash_typ),
				aws.String(ValueType),
			},

			Limit:         aws.Int64(scan_len),
			Segment:       aws.Int64(scan_idx),
			TotalSegments: aws.Int64(scan_all),
		}

		if last_key != nil {
			params.ExclusiveStartKey = last_key
		}

		output, err := d.client.Scan(params)
		if err != nil {
			ticker := backoff.NewTicker(b)
			output, err = d.rescan(ticker, params)
		}

		if err != nil {
			return err
		}

		for _, item := range output.Items {
			all++
			key := GetItemValue(item[hash_typ])
			value := GetItemValue(item[ValueType])

			channel <- DynamoKV{
				key,
				value,
			}
		}

		last_key = output.LastEvaluatedKey
		if last_key == nil {
			break
		}

	}

	return nil
}

//修改DynamoDB预置写入吞吐量
func (d *DynamoDB) UpdateWriteCapacityUnits(table_name string, writeCapacityUnits int64) error {
	readCapacityUnits, oldWriteCapacityUnits, err := d.QueryThroughput(table_name)
	if err != nil {
		return err
	}
	if oldWriteCapacityUnits == writeCapacityUnits {
		return nil
	}

	input := &DDB.UpdateTableInput{
		TableName: aws.String(table_name),
		ProvisionedThroughput: &DDB.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(readCapacityUnits),
			WriteCapacityUnits: aws.Int64(writeCapacityUnits),
		},
	}
	_, err = d.client.UpdateTable(input)
	if err != nil {
		return err
	}
	return nil
}

//查询DynamoDB预置读写吞吐量
func (d *DynamoDB) QueryThroughput(table_name string) (int64, int64, error) {
	inputPre := &DDB.DescribeTableInput{
		TableName: aws.String(table_name),
	}
	outputPre, err := d.client.DescribeTable(inputPre)
	if err != nil {
		return 0, 0, err
	}
	return *outputPre.Table.ProvisionedThroughput.ReadCapacityUnits,
		*outputPre.Table.ProvisionedThroughput.WriteCapacityUnits,
		nil
}
