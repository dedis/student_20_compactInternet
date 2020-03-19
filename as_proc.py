# Some util functions to process and displayAS graph daita from CAIDA

# Relation types to IDs conversion (adopted to numerically manipulate numpy arrays):
#    - peer-peer:         0
#    - provider-customer:-1
#    - customer-provider: 1

import pandas as pd
import numpy as np
    

def extract_links(filename):
    raw_data = pd.read_csv(filename, delimiter='\n', comment='#', header=None, encoding='ISO-8859-1')
    raw_data['provider_peer'], raw_data['customer_peer'], raw_data['relation_type'], raw_data['source'] = raw_data[0].str.split('|').str
    raw_data['relation_type']  = raw_data['relation_type'].apply(lambda x: int(x))
    links = raw_data[['provider_peer','customer_peer','relation_type']]
    return links

def get_nodes_edges(links):
    nodes = np.unique(links['provider_peer'].values.astype(int))
    nodes = np.unique(np.concatenate([nodes, np.unique(links['customer_peer'].values.astype(int))]))
    # Convert AS labels to int
    a_to_b_edges = links.values.astype(int)
    # Invert the edges' endpoints
    b_to_a_edges = a_to_b_edges[:, [1,0,2]]
    # Update the relations
    b_to_a_edges[:,2] = (-1) * b_to_a_edges[:,2]
    # Generate the undirected graph
    edges = np.concatenate((a_to_b_edges, b_to_a_edges), axis=0)
    return (nodes, edges[np.argsort(edges[:,0])])

def node_degree_distribution(nodes, edges):
    _, nd_deg = np.unique(edges[:,0], return_counts=True)
    deg_val, deg_freq = np.unique(np.array(nd_deg), return_counts=True)
    nodes_num = float(len(nodes))
    norm_freq = deg_freq / nodes_num
    return np.concatenate((np.array(deg_val)[..., np.newaxis], np.array(norm_freq)[..., np.newaxis], np.array(deg_freq)[..., np.newaxis]), axis=1)

def load_graph(filename):
    lnks = extract_links(filename)
    (nd, ed) = get_nodes_edges(lnks)
    deg_distr = node_degree_distribution(nd, ed)
    return {'nodes': nd, 'edges': ed, 'deg_distr': deg_distr}

def display_graph(graph, year_label, plt_handle):
    plt_handle.scatter(graph['deg_distr'][:,0], graph['deg_distr'][:,1], label=str(year_label))
