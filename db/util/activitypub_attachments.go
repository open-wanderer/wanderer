package util

import pub "github.com/go-ap/activitypub"

// ActivityPubAttachmentArray keeps even a single attachment in a JSON array.
// Older peers require an array when importing GPX, photos and list avatars.
func ActivityPubAttachmentArray(items pub.ItemCollection) pub.Item {
	if len(items) == 0 {
		return nil
	}
	array := attachmentArray(items)
	return &array
}

// The underlying slice also remains convertible to *pub.ItemCollection for
// code that inspects an outgoing object before its wire serialization.
type attachmentArray pub.ItemCollection

func (a attachmentArray) GetID() pub.ID                       { return pub.EmptyID }
func (a attachmentArray) GetLink() pub.IRI                    { return pub.EmptyIRI }
func (a attachmentArray) GetType() pub.ActivityVocabularyType { return pub.CollectionOfItems }
func (a attachmentArray) IsLink() bool                        { return false }
func (a attachmentArray) IsObject() bool                      { return false }
func (a attachmentArray) IsCollection() bool                  { return true }

func (a attachmentArray) MarshalJSON() ([]byte, error) {
	data := []byte{}
	if !pub.JSONWriteItemCollectionValue(&data, pub.ItemCollection(a), false) {
		return []byte("[]"), nil
	}
	return data, nil
}

// ActivityStreams also permits a bare attachment object. Accept it from peers
// whose serializer compacts one-element collections, as well as normal arrays.
func ActivityPubAttachmentItems(item pub.Item) (*pub.ItemCollection, error) {
	if pub.IsNil(item) {
		empty := pub.ItemCollection{}
		return &empty, nil
	}
	if item.IsCollection() {
		return pub.ToItemCollection(item)
	}
	items := pub.ItemCollection{item}
	return &items, nil
}
