package inssqs

import (
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go/middleware"
)

type entry interface {
	getId() *string
}

type SQSMessageEntry struct {
	Id                     *string
	MessageBody            *string
	DelaySeconds           int32
	MessageDeduplicationId *string
	MessageGroupId         *string
}

func (e SQSMessageEntry) getId() *string {
	return e.Id
}

func (e SQSMessageEntry) toSendMessageBatchRequestEntry() types.SendMessageBatchRequestEntry {
	return types.SendMessageBatchRequestEntry{
		Id:                     e.Id,
		MessageBody:            e.MessageBody,
		DelaySeconds:           e.DelaySeconds,
		MessageDeduplicationId: e.MessageDeduplicationId,
		MessageGroupId:         e.MessageGroupId,
	}
}

type SQSDeleteMessageEntry struct {
	Id            *string
	ReceiptHandle *string
}

func (e SQSDeleteMessageEntry) getId() *string {
	return e.Id
}

func (e SQSDeleteMessageEntry) toDeleteMessageBatchRequestEntry() types.DeleteMessageBatchRequestEntry {
	return types.DeleteMessageBatchRequestEntry{
		Id:            e.Id,
		ReceiptHandle: e.ReceiptHandle,
	}
}

type ReceiveMessageOutput struct {
	// A list of messages.
	Messages []types.Message

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata
}

type ReceiveMessageInput struct {

	// The URL of the Amazon SQS queue from which messages are received. Queue URLs
	// and names are case-sensitive.
	//
	// This member is required.
	QueueUrl *string

	// A list of attributes that need to be returned along with each message. These
	// attributes include:
	//   - All – Returns all values.
	//   - ApproximateFirstReceiveTimestamp – Returns the time the message was first
	//   received from the queue ( epoch time (http://en.wikipedia.org/wiki/Unix_time)
	//   in milliseconds).
	//   - ApproximateReceiveCount – Returns the number of times a message has been
	//   received across all queues but not deleted.
	//   - AWSTraceHeader – Returns the X-Ray trace header string.
	//   - SenderId
	//   - For a user, returns the user ID, for example ABCDEFGHI1JKLMNOPQ23R .
	//   - For an IAM role, returns the IAM role ID, for example
	//   ABCDE1F2GH3I4JK5LMNOP:i-a123b456 .
	//   - SentTimestamp – Returns the time the message was sent to the queue ( epoch
	//   time (http://en.wikipedia.org/wiki/Unix_time) in milliseconds).
	//   - SqsManagedSseEnabled – Enables server-side queue encryption using SQS owned
	//   encryption keys. Only one server-side encryption option is supported per queue
	//   (for example, SSE-KMS (https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-configure-sse-existing-queue.html)
	//   or SSE-SQS (https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-configure-sqs-sse-queue.html)
	//   ).
	//   - MessageDeduplicationId – Returns the value provided by the producer that
	//   calls the SendMessage action.
	//   - MessageGroupId – Returns the value provided by the producer that calls the
	//   SendMessage action. Messages with the same MessageGroupId are returned in
	//   sequence.
	//   - SequenceNumber – Returns the value provided by Amazon SQS.
	AttributeNames []types.QueueAttributeName

	// The maximum number of messages to return. Amazon SQS never returns more
	// messages than this value (however, fewer messages might be returned). Valid
	// values: 1 to 10. Default: 1.
	MaxNumberOfMessages int32

	// The name of the message attribute, where N is the index.
	//   - The name can contain alphanumeric characters and the underscore ( _ ),
	//   hyphen ( - ), and period ( . ).
	//   - The name is case-sensitive and must be unique among all attribute names for
	//   the message.
	//   - The name must not start with AWS-reserved prefixes such as AWS. or Amazon.
	//   (or any casing variants).
	//   - The name must not start or end with a period ( . ), and it should not have
	//   periods in succession ( .. ).
	//   - The name can be up to 256 characters long.
	// When using ReceiveMessage , you can send a list of attribute names to receive,
	// or you can return all of the attributes by specifying All or .* in your
	// request. You can also use all message attributes starting with a prefix, for
	// example bar.* .
	MessageAttributeNames []string

	// This parameter applies only to FIFO (first-in-first-out) queues. The token used
	// for deduplication of ReceiveMessage calls. If a networking issue occurs after a
	// ReceiveMessage action, and instead of a response you receive a generic error, it
	// is possible to retry the same action with an identical ReceiveRequestAttemptId
	// to retrieve the same set of messages, even if their visibility timeout has not
	// yet expired.
	//   - You can use ReceiveRequestAttemptId only for 5 minutes after a
	//   ReceiveMessage action.
	//   - When you set FifoQueue , a caller of the ReceiveMessage action can provide a
	//   ReceiveRequestAttemptId explicitly.
	//   - If a caller of the ReceiveMessage action doesn't provide a
	//   ReceiveRequestAttemptId , Amazon SQS generates a ReceiveRequestAttemptId .
	//   - It is possible to retry the ReceiveMessage action with the same
	//   ReceiveRequestAttemptId if none of the messages have been modified (deleted or
	//   had their visibility changes).
	//   - During a visibility timeout, subsequent calls with the same
	//   ReceiveRequestAttemptId return the same messages and receipt handles. If a
	//   retry occurs within the deduplication interval, it resets the visibility
	//   timeout. For more information, see Visibility Timeout (https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-visibility-timeout.html)
	//   in the Amazon SQS Developer Guide. If a caller of the ReceiveMessage action
	//   still processes messages when the visibility timeout expires and messages become
	//   visible, another worker consuming from the same queue can receive the same
	//   messages and therefore process duplicates. Also, if a consumer whose message
	//   processing time is longer than the visibility timeout tries to delete the
	//   processed messages, the action fails with an error. To mitigate this effect,
	//   ensure that your application observes a safe threshold before the visibility
	//   timeout expires and extend the visibility timeout as necessary.
	//   - While messages with a particular MessageGroupId are invisible, no more
	//   messages belonging to the same MessageGroupId are returned until the
	//   visibility timeout expires. You can still receive messages with another
	//   MessageGroupId as long as it is also visible.
	//   - If a caller of ReceiveMessage can't track the ReceiveRequestAttemptId , no
	//   retries work until the original visibility timeout expires. As a result, delays
	//   might occur but the messages in the queue remain in a strict order.
	// The maximum length of ReceiveRequestAttemptId is 128 characters.
	// ReceiveRequestAttemptId can contain alphanumeric characters ( a-z , A-Z , 0-9 )
	// and punctuation ( !"#$%&'()*+,-./:;<=>?@[\]^_`{|}~ ). For best practices of
	// using ReceiveRequestAttemptId , see Using the ReceiveRequestAttemptId Request
	// Parameter (https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/using-receiverequestattemptid-request-parameter.html)
	// in the Amazon SQS Developer Guide.
	ReceiveRequestAttemptId *string

	// The duration (in seconds) that the received messages are hidden from subsequent
	// retrieve requests after being retrieved by a ReceiveMessage request.
	VisibilityTimeout int32

	// The duration (in seconds) for which the call waits for a message to arrive in
	// the queue before returning. If a message is available, the call returns sooner
	// than WaitTimeSeconds . If no messages are available and the wait time expires,
	// the call returns successfully with an empty list of messages. To avoid HTTP
	// errors, ensure that the HTTP response timeout for ReceiveMessage requests is
	// longer than the WaitTimeSeconds parameter. For example, with the Java SDK, you
	// can set HTTP transport settings using the NettyNioAsyncHttpClient (https://sdk.amazonaws.com/java/api/latest/software/amazon/awssdk/http/nio/netty/NettyNioAsyncHttpClient.html)
	// for asynchronous clients, or the ApacheHttpClient (https://sdk.amazonaws.com/java/api/latest/software/amazon/awssdk/http/apache/ApacheHttpClient.html)
	// for synchronous clients.
	WaitTimeSeconds int32
}