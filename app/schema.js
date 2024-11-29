import { gql } from "apollo-server-core";

export const schema = gql`
    scalar JSON
    scalar DateTime
    scalar URL

    type Query{
        weather(lat: String!, lng: String!): JSON
        weatherEstonia(lat: String!, lng: String!): JSON
    }
`;
